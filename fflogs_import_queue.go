package main

import (
	"log"
	"sync"
	"time"
)

type FFLogsImportQueue struct {
	lock    sync.Mutex
	reports []string
	db      *DatabaseHandler
	fflog   *FFLogsHandler
}

func NewFFLogsImportQueue(config *Config, db *DatabaseHandler) (*FFLogsImportQueue, error) {
	fflogHandler, err := NewFFLogsHandler(config)
	if err != nil {
		return nil, err
	}
	return &FFLogsImportQueue{
		reports: make([]string, 0),
		db:      db,
		fflog:   fflogHandler,
	}, nil
}

func (f *FFLogsImportQueue) Add(reportID string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for _, existingReportID := range f.reports {
		if existingReportID == reportID {
			return ErrAlreadyInQueue
		}
	}
	f.reports = append(f.reports, reportID)
	log.Printf("Added FFLogs report %s to queue.\n", reportID)
	return nil
}

func (f *FFLogsImportQueue) Start() {
	defer f.lock.Unlock()
	for range time.Tick(time.Second * 1) {
		f.lock.Lock()
		if len(f.reports) == 0 {
			f.lock.Unlock()
			continue
		}
		reportID := ""
		reportID, f.reports = f.reports[0], f.reports[1:]
		log.Printf("Processing FFLogs report %s.\n", reportID)
		reportFights, err := f.fflog.FetchReportFights(reportID)
		if err != nil {
			log.Printf("Error importing FFLogs report %s: %s\n", reportID, err.Error())
			f.lock.Unlock()
			continue
		}
		if err := f.db.HandleFFLogsReportFights(reportID, reportFights); err != nil {
			log.Printf("Error importing FFLogs report %s: %s\n", reportID, err.Error())
		}
		f.lock.Unlock()
	}
}
