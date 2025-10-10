package repository

import "time"

func StartStorageBackup(storage Storage, interval time.Duration, filename string) func() {
	if memStorage, ok := storage.(*memStorage); ok {
		memStorage.StartPeriodicSave(interval, filename)
		return func() {
			memStorage.StopPeriodicSave()
		}
	}
	return func() {}
}
