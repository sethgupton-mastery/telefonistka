package telefonistka

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wayfair-incubator/telefonistka/internal/pkg/githubapi"
)

// This is still(https://github.com/spf13/cobra/issues/1862) the documented way to use cobra
func init() { //nolint:gochecknoinits
	var eventType string
	var eventFilePath string
	eventCmd := &cobra.Command{
		Use:   "event",
		Short: "Handles a GitHub event based on event JSON file",
		Long:  "Handles a GitHub event based on event JSON file.\nThis operation mode was was built with GitHub Actions in mind",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			event(eventType, eventFilePath)
		},
	}
	eventCmd.Flags().StringVarP(&eventType, "type", "t", getEnv("GITHUB_EVENT_NAME", ""), "Event type, defaults to GITHUB_EVENT_NAME env var")
	eventCmd.Flags().StringVarP(&eventFilePath, "file", "f", getEnv("GITHUB_EVENT_PATH", ""), "File path for event JSON, defaults to GITHUB_EVENT_PATH env var")
	rootCmd.AddCommand(eventCmd)
}

func event(eventType string, eventFilePath string) {
	ctx := context.Background()

	log.Infof("Event type: %s", eventType)
	log.Infof("Proccesing file: %s", eventFilePath)

	payload, err := os.ReadFile(eventFilePath)
	if err != nil {
		panic(err)
	}

	mainGithubClient, githubGraphQlClient, prApproverGithubClient := githubapi.CreateAllClients(ctx)
	botIdentity, _ := githubapi.GetBotGhIdentity(githubGraphQlClient, ctx)
	githubapi.HandleEvent(eventType, payload, mainGithubClient, prApproverGithubClient, githubGraphQlClient, ctx, botIdentity)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
