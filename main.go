package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "bind-address",
				Value: ":3001",
				EnvVars: []string{
					"BIND_ADDRESS",
				},
			},
		},
		Action: func(c *cli.Context) error {

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			for _, e := range os.Environ() {
				// os.Getenv
				parts := strings.SplitN(e, "=", 2)
				if len(parts) != 2 {
					return errors.Errorf("malformed env %q", e)
				}

				name, value := parts[0], parts[1]

				prefix := "WATCH_"

				if strings.HasPrefix(name, prefix) {
					targetName := name[len(prefix):]
					url := value
					watchURL(targetName, url, logger.Sugar().With("targetName", targetName, "targetUrl", url))
				}
			}

			http.Handle("/metrics", promhttp.Handler())
			a := c.String("bind-address")
			logger.Sugar().Infof("Listening on %s", a)
			return http.ListenAndServe(a, nil)
		},
	}

	app.RunAndExitOnError()

}

func watchURL(name, target string, logger *zap.SugaredLogger) error {
	_, err := url.Parse(target)
	if err != nil {
		return errors.Wrapf(err, "while parsing URL %q", target)
	}
	statusCodeGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "critic_target_status_code",
		ConstLabels: prometheus.Labels{
			"name": name,
			"url":  target,
		},
	})

	logger.Infof("watching %s: %s", name, target)
	go func() {
		for ; ; time.Sleep(30 * time.Second) {

			req, err := http.NewRequest("GET", target, nil)
			if err != nil {
				logger.With("error", err).Error("while creating http request")
				statusCodeGauge.Set(0.0)
				continue
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.With("error", err).Error("while performing http request")
				statusCodeGauge.Set(1.0)
				continue
			}

			statusCodeGauge.Set(float64(res.StatusCode))

			defer res.Body.Close()

		}

	}()
	return nil
}

var x = promauto.NewGauge(prometheus.GaugeOpts{
	// Namespace: "critic",
	// Subsystem: "Namespace",
	Name: "x",

	// ConstLabels:
})
