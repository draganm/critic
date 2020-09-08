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
	u, err := url.Parse(target)
	if err != nil {
		return errors.Wrapf(err, "while parsing URL %q", target)
	}

	isHTTPS := u.Scheme == "https"

	var serverCertificateExpirationTime prometheus.Gauge

	if isHTTPS {
		serverCertificateExpirationTime = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "critic_target_server_certificate_expiration_time",
			ConstLabels: prometheus.Labels{
				"name": name,
				"url":  target,
			},
		})
	}

	requestDuration := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "critic_target_request_duration",
		ConstLabels: prometheus.Labels{
			"name": name,
			"url":  target,
		},
	})

	statusCodeGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "critic_target_status_code",
		ConstLabels: prometheus.Labels{
			"name": name,
			"url":  target,
		},
	})

	probeFailedCounter := promauto.NewCounter(prometheus.CounterOpts{
		Name: "critic_target_probe_failed_counter",
		ConstLabels: prometheus.Labels{
			"name": name,
			"url":  target,
		},
	})

	targetIsHealthyGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "critic_target_is_healthy",
		ConstLabels: prometheus.Labels{
			"name": name,
			"url":  target,
		},
	})

	logger.Infof("watching %s: %s", name, target)
	go func() {

		for ; ; time.Sleep(30 * time.Second) {

			startTime := time.Now()

			req, err := http.NewRequest("GET", target, nil)
			if err != nil {
				logger.With("error", err).Error("while creating http request")
				statusCodeGauge.Set(0.0)
				probeFailedCounter.Add(1)
				targetIsHealthyGauge.Set(0)
				duration := time.Since(startTime)
				requestDuration.Set(duration.Seconds())
				continue
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.With("error", err).Error("while performing http request")
				statusCodeGauge.Set(1.0)
				probeFailedCounter.Add(1)
				targetIsHealthyGauge.Set(0)
				duration := time.Since(startTime)
				requestDuration.Set(duration.Seconds())
				continue
			}

			var certExpiryTime = 0.0

			if isHTTPS {
				if res.TLS != nil {
					certs := res.TLS.PeerCertificates
					if len(certs) > 0 {
						cert := certs[0]
						certExpiryTime = float64(cert.NotAfter.Unix())
					}
				}

				serverCertificateExpirationTime.Set(certExpiryTime)
			}

			duration := time.Since(startTime)
			requestDuration.Set(duration.Seconds())

			statusCodeGauge.Set(float64(res.StatusCode))

			failed := res.StatusCode < 100 || res.StatusCode >= 500 || res.StatusCode == 404

			if failed {
				probeFailedCounter.Add(1)
				targetIsHealthyGauge.Set(0)
			} else {
				targetIsHealthyGauge.Set(1)
			}

			res.Body.Close()

		}

	}()
	return nil
}
