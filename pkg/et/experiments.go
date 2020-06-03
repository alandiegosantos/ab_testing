package et

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Experiment struct {
	Name     string
	Versions []string
}

// This is our database of experiments.
// TODO: Migrate this to a persistent storage
var experimentsDb sync.Map

var (
	experimentsCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "experiments_count",
			Help: "The total number of experiments",
		},
		[]string{"experiment", "version"},
	)

	conversionCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "conversion_count",
			Help: "The total number of conversion",
		},
		[]string{"experiment", "version"},
	)
)

func init() {

	// Adding two experiments

	experimentsDb.Store(
		"button_color",
		&Experiment{
			Name: "button_color",
			Versions: []string{
				"red",
				"blue",
				"green",
			},
		},
	)

	experimentsDb.Store(
		"title_text",
		&Experiment{
			Name: "title_text",
			Versions: []string{
				"Showing version 1",
				"This is version 2",
				"Version 3. It is awesome!",
			},
		},
	)

	rand.Seed(time.Now().Unix())

}

func NewExperiment(name string, start, end time.Time, versions []string) error {

	experiment := &Experiment{
		Name:     name,
		Versions: versions,
	}

	experimentsDb.Store(name, experiment)

	return nil

}

func GetExperiments() []*Experiment {

	experiments := make([]*Experiment, 0, 0)

	experimentsDb.Range(func(k, v interface{}) bool {

		experiments = append(experiments, v.(*Experiment))

		return true

	})

	return experiments

}

func GetExperimentValue(name string) (string, error) {

	exp, ok := experimentsDb.Load(name)

	if !ok {
		return "", fmt.Errorf("Experiment %s do not found", name)
	}

	experiment := exp.(*Experiment)

	index := rand.Intn(len(experiment.Versions))

	return experiment.Versions[index], nil

}

func GetExperimentValues() map[string]string {

	experimentValues := make(map[string]string)

	experimentsDb.Range(func(k, v interface{}) bool {

		experiment := v.(*Experiment)

		index := rand.Intn(len(experiment.Versions))

		experimentsCount.With(
			prometheus.Labels{
				"experiment": experiment.Name,
				"version":    strconv.Itoa(index),
			},
		).Inc()

		experimentValues[experiment.Name] = experiment.Versions[index]

		return true

	})

	return experimentValues

}

func IncConversionCounter(experimentName, version string) {

	e, ok := experimentsDb.Load(experimentName)

	if ok {

		exp := e.(*Experiment)

		for index, v := range exp.Versions {

			if strings.Compare(v, version) == 0 {

				conversionCount.With(
					prometheus.Labels{
						"experiment": exp.Name,
						"version":    strconv.Itoa(index),
					},
				).Inc()

			}

		}

	}

}
