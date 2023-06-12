package telefonistka

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/alexliesenfeld/health"
	"github.com/google/go-github/v52/github"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/wayfair-incubator/telefonistka/internal/pkg/githubapi"
	prom "github.com/wayfair-incubator/telefonistka/internal/pkg/prometheus"
)

func getCrucialEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Fatalf("%s environment variable is required", key)
	os.Exit(3)
	return ""
}

var serveCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs the web server that listens to GitHub webhooks",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

// This is still(https://github.com/spf13/cobra/issues/1862) the documented way to use cobra
func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(serveCmd)
}

func handleWebhook(ctx context.Context, githubWebhookSecret []byte, mainGhClientCache *lru.Cache[string, githubapi.GhClientPair], prApproverGhClientCache *lru.Cache[string, githubapi.GhClientPair]) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		// payload, err := ioutil.ReadAll(r.Body)
		payload, err := github.ValidatePayload(r, githubWebhookSecret)
		if err != nil {
			log.Errorf("error reading request body: err=%s\n", err)
			prom.InstrumentWebhookHit("validation_failed")
			return
		}
		eventType := github.WebHookType(r)

		githubapi.HandleEvent(eventType, payload, ctx, mainGhClientCache, prApproverGhClientCache)
	}
}

func serve() {
	ctx := context.Background()
	githubWebhookSecret := []byte(getCrucialEnv("GITHUB_WEBHOOK_SECRET"))
	livenessChecker := health.NewChecker() // No checks for the moment, other then the http server availability
	readinessChecker := health.NewChecker()

	// mainGhClientCache := map[string]githubapi.GhClientPair{} //GH apps use a per-account/org client
	mainGhClientCache, _ := lru.New[string, githubapi.GhClientPair](128)
	prApproverGhClientCache, _ := lru.New[string, githubapi.GhClientPair](128)

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", handleWebhook(ctx, githubWebhookSecret, mainGhClientCache, prApproverGhClientCache))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/live", health.NewHandler(livenessChecker))
	mux.Handle("/ready", health.NewHandler(readinessChecker))

	srv := &http.Server{
		Handler:      mux,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Infoln("server started")
	log.Fatal(srv.ListenAndServe())
}
