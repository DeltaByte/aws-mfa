package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	cfg := LoadConfig()
	creds := LoadCredentials()

	// read flags
	var srcProfileName, dstProfileName, device, code string
	var ttl int
	var overwrite bool
	flag.StringVar(&srcProfileName, "src", "", "Source AWS Profile")
	flag.StringVar(&dstProfileName, "dst", "", "Destination AWS Profile")
	flag.StringVar(&device, "device", "", "MFA Device ID")
	flag.StringVar(&code, "code", "", "MFA code")
	flag.IntVar(&ttl, "ttl", 0, "Session token lifetime")
	flag.BoolVar(&overwrite, "overwrite", false, "Overwrite exiting destination profile")
	flag.Parse()

	// source aws profile
	if srcProfileName == "" {
		srcProfileName = prompter.Choose("Source profile name", creds.Names, cfg.PreviousSourceProfile)
	}
	previousProfileConfig, previousConfigExists := cfg.SourceProfiles[srcProfileName]
	_, srcProfileExists := creds.Profiles[srcProfileName]
	if !srcProfileExists {
		log.Fatalf("Unknown profile: %s\n", srcProfileName)
	}

	// destination aws profile
	if dstProfileName == "" {
		createNewProfile := prompter.YesNo("Create a new destination profile?", !previousConfigExists)

		// use existing profile
		if !createNewProfile {
			dstProfileName = prompter.Choose("Destination profile name", creds.Names, previousProfileConfig.DestinationProfile)
		}

		// create new profile
		if createNewProfile {
			dstProfileName = prompter.Prompt("New profile name", fmt.Sprintf("%s-mfa", srcProfileName))
			creds.New(dstProfileName)
		}
	}
	dstProfile, ok := creds.Profiles[dstProfileName]
	if !ok {
		log.Fatalf("Unknown profile: %s\n", dstProfileName)
	}

	// MFA device
	if device == "" {
		device = prompter.Prompt("MFA device ID", previousProfileConfig.MfaDevice)
	}

	// only support TOTP
	u2fArnRegex := regexp.MustCompile(`arn:aws:iam::\d+:u2f\/.+`)
	if u2fArnRegex.MatchString(device) {
		log.Fatalln("Due to a limitation in AWS STS, U2F MFA devices are not supported. Use a TOTP app instead.")
	}

	// TTL
	if ttl == 0 {
		defaultTTL := 3600
		if previousProfileConfig.TTL > 0 {
			defaultTTL = previousProfileConfig.TTL
		}
		ttlInput, err := strconv.Atoi(prompter.Prompt("Session TTL", strconv.Itoa(defaultTTL)))
		if err != nil {
			log.Fatalln("Failed to parse TTL", err)
		}
		ttl = ttlInput
	}

	// save config
	cfg.PreviousMFADevice = device
	cfg.PreviousSourceProfile = srcProfileName
	cfg.SourceProfiles[srcProfileName] = SourceProfile{
		Name:               srcProfileName,
		MfaDevice:          device,
		DestinationProfile: dstProfileName,
		TTL:                ttl,
	}
	if err := cfg.Save(); err != nil {
		log.Fatalln("Failed to save config")
	}

	// get MFA code
	if code == "" {
		code = prompter.Prompt("MFA Code", "")
	}

	// init aws sdk
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(srcProfileName))
	if err != nil {
		log.Fatalln("Failed to load AWS config")
	}
	awsConfig.Region = "aws-global"

	// get sts session
	stsClient := sts.NewFromConfig(awsConfig)
	session, err := stsClient.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(int32(ttl)),
		SerialNumber:    aws.String(device),
		TokenCode:       aws.String(code),
	})
	if err != nil {
		log.Fatalln("Failed to create STS session", err)
	}

	// store creds
	if !dstProfile.isNew && !overwrite {
		overwrite := prompter.YesNo("Overwrite existing AWS profile?", true)
		if !overwrite {
			log.Fatalln("Aborting...")
		}
	}
	dstProfile.AccessKeyId = *session.Credentials.AccessKeyId
	dstProfile.SecretAccessKey = *session.Credentials.SecretAccessKey
	dstProfile.SessionToken = *session.Credentials.SessionToken
	dstProfile.isChanged = true
	creds.Profiles[dstProfileName] = dstProfile
	if err := creds.Save(); err != nil {
		log.Fatalln("Failed to save credentials", err)
	}

	log.Printf("Session created! Expires: %s", (*session.Credentials.Expiration).Format(time.RFC1123Z))
}
