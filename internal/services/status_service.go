package services

import (
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/pecigonzalo/kafka-canary/internal/canary"
	"github.com/pecigonzalo/kafka-canary/internal/services/util"
)

// Status defines useful status related information
type Status struct {
	Consuming ConsumingStatus
}

// ConsumingStatus defines consuming related status information
type ConsumingStatus struct {
	TimeWindow time.Duration
	Percentage float64
}

type statusService struct {
	canaryConfig           *canary.Config
	producedRecordsSamples util.TimeWindowRing
	consumedRecordsSamples util.TimeWindowRing
	logger                 *zerolog.Logger
}

func NewStatusServiceService(canary canary.Config, logger *zerolog.Logger) StatusService {
	return &statusService{
		canaryConfig: &canary,
		logger:       logger,
	}
}

func (s *statusService) Open()  {}
func (s *statusService) Close() {}
func (s *statusService) StatusHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		status := Status{}

		// update consuming related status section
		status.Consuming = ConsumingStatus{
			TimeWindow: s.canaryConfig.StatusCheckInterval * time.Duration(s.consumedRecordsSamples.Count()),
		}
		consumedPercentage, err := s.consumedPercentage()
		if e, ok := err.(*util.ErrNoDataSamples); ok {
			status.Consuming.Percentage = -1
			s.logger.Error().Err(err).Msgf("Error processing consumed records percentage: %v", e)
		} else {
			status.Consuming.Percentage = consumedPercentage
		}

		json, err := json.Marshal(status)
		if err != nil {
			s.logger.Error().Err(err).Msg("Marshal status")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Header().Add("Content-Type", "application/json")
		_, err = rw.Write(json)
		if err != nil {
			s.logger.Err(err).Msg("Write response")
		}
	})
}

// consumedPercentage function processes the percentage of consumed messages in the specified time window
func (s *statusService) consumedPercentage() (float64, error) {
	// sampling for produced (and consumed records) not done yet
	if s.producedRecordsSamples.IsEmpty() {
		return 0, &util.ErrNoDataSamples{}
	}

	// get number of records consumed and produced since the beginning of the time window (tail of ring buffers)
	consumed := s.consumedRecordsSamples.Head() - s.consumedRecordsSamples.Tail()
	produced := s.producedRecordsSamples.Head() - s.producedRecordsSamples.Tail()

	if produced == 0 {
		return 0, &util.ErrNoDataSamples{}
	}

	percentage := float64(consumed*100) / float64(produced)
	// rounding to two decimal digits
	percentage = math.Round(percentage*100) / 100
	s.logger.Info().Msgf("Status consumed percentage = %f", percentage)
	return percentage, nil
}
