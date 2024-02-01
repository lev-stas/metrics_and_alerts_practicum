package routers

import (
	"database/sql"
	"github.com/go-resty/resty/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lev-stas/metricsmonitor.git/internal/configs"
	"github.com/lev-stas/metricsmonitor.git/internal/metricsstorage"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRootRouter(t *testing.T) {
	storage := metricsstorage.NewMemStorage()
	storage.Set("TestGauge", 3.14)
	storage.Inc("TestCounter", 88)
	tempFile, err := os.CreateTemp("", "test_metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())
	configs.ServerParams.StorageFile = tempFile.Name()
	fileWriter, err := metricsstorage.NewFileWriter(&configs.ServerParams)
	if err != nil {
		t.Fatal(err)
	}
	defer fileWriter.Close()

	db, err := sql.Open("pgx", configs.ServerParams.DBConnect)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := httptest.NewServer(RootRouter(storage, fileWriter, db))
	client := resty.New()
	updateUrl := ts.URL + "/update/"
	valueUrl := ts.URL + "/value/"

	var testCasesWithBody = []struct {
		testName     string
		method       string
		requestBody  string
		expectedBody string
		status       int
	}{
		{
			"Test post valid gauge request with body",
			http.MethodPost,
			`{
				"id": "GaugeMetric",
				"type": "gauge",
				"value": 3.14
			}`,
			`{
							"id": "GaugeMetric",
							"type": "gauge",
							"value": 3.14
						}`,

			200,
		},
		{
			"Test post valid counter request with body",
			http.MethodPost,
			`{
					"id": "CounterMetric",
					"type": "counter",
					"delta": 3
					}`,
			`{
							"id": "CounterMetric",
							"type": "counter",
							"delta": 3
								}`,
			200,
		},
		{
			"Test post no metric name request with body",
			http.MethodPost,
			`{
					"type": "gauge",
					"value": 3.14
					}`,
			"",
			404,
		},
		{
			"Test post wrong metric type request with body",
			http.MethodPost,
			`{
					"id": "WrongMetric",
					"type": "wrongtype",
					"value": 3.12
					}`,
			"",
			400,
		},
		{
			"Test post wrong gauge type request with body",
			http.MethodPost,
			`{
						"id": "WrongMetric",
						"type": "gauge",
						"value": "abc"
					}`,
			"",
			400,
		},
		{
			"Test post wrong counter type request with body",
			http.MethodPost,
			`{
					"id": "WrongMetric",
					"type": "counter",
					"delta": 3.14
					}`,
			"",
			400,
		},
	}
	for _, v := range testCasesWithBody {
		t.Run(v.testName, func(t *testing.T) {
			request := resty.New().R()
			request.URL = updateUrl
			request.Method = v.method

			if len(v.requestBody) > 0 {
				request.SetHeader("Content-Type", "application/json")
				request.SetBody(v.requestBody)
			}
			resp, err := request.Send()
			assert.NoError(t, err)
			assert.Equal(t, v.status, resp.StatusCode())
			if v.expectedBody != "" {
				assert.JSONEq(t, v.expectedBody, string(resp.Body()))
			}
		})
	}

	var testCases = []struct {
		testName string
		url      string
		status   int
	}{
		{"Test post valid gauge request", "/update/gauge/GaugeMetric/3.14", 200},
		{"Test post valid counter request", "/update/counter/CounterMetric/3", 200},
		{"Test post no metric name request", "/update/gauge//435", 404},
		{"Test post no metric valid request", "/update/gauge/MetricName", 404},
		{"Test post wrong metric type request", "/update/wrongType/WrongMetric/435", 400},
		{"Test post wrong gauge type request", "/update/gauge/WrongMetric/abc", 400},
		{"Test post wrong counter type request", "/update/counter/WrongMetric/3.14", 400},
	}
	for _, v := range testCases {
		t.Run(v.testName, func(t *testing.T) {
			resp, err := client.R().Post(ts.URL + v.url)
			assert.NoError(t, err)
			assert.Equal(t, v.status, resp.StatusCode())
		})
	}

	t.Run("Wrong request method for update endpoint", func(t *testing.T) {
		resp, err := client.R().Get(updateUrl)
		assert.NoError(t, err)
		assert.Equal(t, 405, resp.StatusCode())
	})

	t.Run("Test main page", func(t *testing.T) {
		resp, err := client.R().Get(ts.URL)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Contains(t, string(resp.Body()), "<h1>Метрики</h1>")
		assert.Contains(t, string(resp.Body()), "TestGauge: 3.14")
		assert.Contains(t, string(resp.Body()), "TestCounter: 88")
	})
	t.Run("Get gauge metric by POST request", func(t *testing.T) {
		resp, err := client.R().SetBody(`{"id": "TestGauge", "type": "gauge"}`).Post(valueUrl)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.JSONEq(t, `{"id": "TestGauge", "type": "gauge", "value": 3.14}`, string(resp.Body()))
	})
	t.Run("Get counter Metrics by POST request", func(t *testing.T) {
		resp, err := client.R().SetBody(`{"id": "TestCounter", "type": "counter"}`).Post(valueUrl)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.JSONEq(t, `{"id": "TestCounter", "type": "counter", "delta": 88}`, string(resp.Body()))
	})
	t.Run("Get wrong metrics type by POST request", func(t *testing.T) {
		resp, err := client.R().SetBody(`{"id": "TestCounter", "type": "wrongType"}`).Post(valueUrl)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode())
	})
	t.Run("Get missed metric by POST request", func(t *testing.T) {
		resp, err := client.R().SetBody(`{"id": "TestGauge", "type": "counter"}`).Post(valueUrl)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
	t.Run("Get gauge metric", func(t *testing.T) {
		resp, err := client.R().Get(ts.URL + "/value/gauge/TestGauge")
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "3.14", string(resp.Body()))
	})
	t.Run("Get counter Metrics", func(t *testing.T) {
		resp, err := client.R().Get(ts.URL + "/value/counter/TestCounter")
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())
		assert.Equal(t, "88", string(resp.Body()))
	})
	t.Run("Get wrong metrics type", func(t *testing.T) {
		resp, err := client.R().Get(ts.URL + "/value/wrongMetrics/TestCounter")
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode())
	})
	t.Run("Get missed metric", func(t *testing.T) {
		resp, err := client.R().Get(ts.URL + "/value/counter/TestGauge")
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode())
	})
	t.Run("Post method", func(t *testing.T) {
		resp, err := client.R().Post(ts.URL + "/value/counter/TestCounter")
		assert.NoError(t, err)
		assert.Equal(t, 405, resp.StatusCode())
	})
}
