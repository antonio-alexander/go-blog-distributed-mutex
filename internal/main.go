package internal

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func employeeConcurrentMutateBenchmark(config *Configuration, chOsSignal chan (os.Signal),
	mutateFx func(goRoutine int) error) error {
	var wg sync.WaitGroup

	start := make(chan struct{})
	stopper := make(chan struct{})
	defer func() {
		select {
		default:
			close(start)
		case <-start:
		}
	}()
	for i := range config.GoRoutines {
		started := make(chan struct{})
		wg.Add(1)
		go func(goRoutine int) {
			defer wg.Done()

			var totalMutations, totalErrors int
			var totalDuration time.Duration

			defer func() {

				average := "-"
				if totalMutations > 0 {
					average = (totalDuration / time.Duration(totalMutations)).String()
				}
				fmt.Printf("go routine [%d]:\n total mutations: %d\n average time: %s\n total errors: %d\n",
					goRoutine,
					totalMutations,
					average,
					totalErrors)
			}()
			tMutate := time.NewTicker(config.MutateInterval)
			defer tMutate.Stop()
			close(started)
			<-start
			for {
				select {
				case <-stopper:
					return
				case <-tMutate.C:
					tStart := time.Now()
					if err := mutateFx(goRoutine); err != nil {
						totalErrors++
						continue
					}
					totalDuration += time.Since(tStart)
					totalMutations++
				}
			}
		}(i)
		<-started
	}
	close(start)
	select {
	case <-time.After(config.DemoDuration):
	case <-chOsSignal:
	}
	close(stopper)
	wg.Wait()
	return nil
}

func employeeConcurrentMutateDemo(config *Configuration, chOsSignal chan (os.Signal),
	mutateFx func(goRoutine, dataInconsistencies int) (int, error)) error {
	var wg sync.WaitGroup

	start := make(chan struct{})
	stopper := make(chan struct{})
	defer func() {
		select {
		default:
			close(start)
		case <-start:
		}
	}()
	for i := range config.GoRoutines {
		started := make(chan struct{})
		wg.Add(1)
		go func(goRoutine int) {
			defer wg.Done()

			var dataInconsistencies, totalErrors,
				totalMutations int
			var err error

			defer func() {
				fmt.Printf("go routine [%d]:\n total mutations: %d\n data inconsistencies: %d\n total errors: %d\n",
					goRoutine, totalMutations, dataInconsistencies, totalErrors)
			}()
			tMutate := time.NewTicker(config.MutateInterval)
			defer tMutate.Stop()
			close(started)
			<-start
			for {
				select {
				case <-stopper:
					return
				case <-tMutate.C:
					dataInconsistencies, err = mutateFx(goRoutine, dataInconsistencies)
					if err != nil {
						totalErrors++
						continue
					}
					totalMutations++
				}
			}
		}(i)
		<-started
	}
	close(start)
	select {
	case <-time.After(config.DemoDuration):
	case <-chOsSignal:
	}
	close(stopper)
	wg.Wait()
	return nil
}

func employeeCurrentMutateWithMutexBenchmark(config *Configuration, db *sql.DB, mu Mutex, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n=============================================")
	fmt.Println("--Benchmarking Concurrent Mutate with Mutex--")
	fmt.Println("=============================================")
	return employeeConcurrentMutateBenchmark(config, chOsSignal, func(goRoutine int) error {
		mu.Lock()
		defer mu.Unlock()

		if _, err := UpdateEmployee(db, employee); err != nil {
			return err
		}
		return nil
	})
}

func employeeCurrentMutateWithRowLockBenchmark(config *Configuration, db *sql.DB, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n================================================")
	fmt.Println("--Benchmarking Concurrent Mutate with Row Lock--")
	fmt.Println("================================================")
	return employeeConcurrentMutateBenchmark(config, chOsSignal, func(goRoutine int) error {
		if _, _, err := UpdateEmployeeWithLock(db, employee); err != nil {
			return err
		}
		return nil
	})
}

func employeeCurrentMutateWithVersionBenchmark(config *Configuration, db *sql.DB, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n===============================================")
	fmt.Println("--Benchmarking Concurrent Mutate with Version--")
	fmt.Println("===============================================")
	return employeeConcurrentMutateBenchmark(config, chOsSignal, func(goRoutine int) error {
		employeeRead, err := ReadEmployee(db, employee.EmailAddress)
		if err != nil {
			return err
		}
		if _, err := UpdateEmployeeWithVersion(db, employee, employeeRead.Version); err != nil {
			return err
		}
		return nil
	})
}

func employeeCurrentMutateNoMutex(config *Configuration, db *sql.DB, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n===========================================")
	fmt.Println("--Testing Concurrent Mutate with no Mutex--")
	fmt.Println("===========================================")
	return employeeConcurrentMutateDemo(config, chOsSignal, func(goRoutine, dataInconsistencies int) (int, error) {
		employeeRead, err := ReadEmployee(db, employee.EmailAddress)
		if err != nil {
			return dataInconsistencies, err
		}
		employeeUpdated, err := UpdateEmployee(db, employee)
		if err != nil {
			return dataInconsistencies, err
		}
		if employeeUpdated.Version != employeeRead.Version+1 {
			dataInconsistencies++
		}
		return dataInconsistencies, nil
	})
}

func employeeCurrentMutateWithMutexDemo(config *Configuration, db *sql.DB, mu Mutex, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n========================================")
	fmt.Println("--Testing Concurrent Mutate with Mutex--")
	fmt.Println("========================================")
	return employeeConcurrentMutateDemo(config, chOsSignal, func(goRoutine, dataInconsistencies int) (int, error) {
		mu.Lock()
		defer mu.Unlock()

		employeeRead, err := ReadEmployee(db, employee.EmailAddress)
		if err != nil {
			return dataInconsistencies, err
		}
		employeeUpdated, err := UpdateEmployee(db, employee)
		if err != nil {
			return dataInconsistencies, err
		}
		if employeeUpdated.Version != employeeRead.Version+1 {
			dataInconsistencies++
		}
		return dataInconsistencies, nil
	})
}

func employeeCurrentMutateWithRowLockDemo(config *Configuration, db *sql.DB, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n===========================================")
	fmt.Println("--Testing Concurrent Mutate with Row Lock--")
	fmt.Println("===========================================")
	return employeeConcurrentMutateDemo(config, chOsSignal, func(goRoutine, dataInconsistencies int) (int, error) {
		employeeRead, employeeUpdated, err := UpdateEmployeeWithLock(db, employee)
		if err != nil {
			return dataInconsistencies, err
		}
		if employeeUpdated.Version != employeeRead.Version+1 {
			dataInconsistencies++
		}
		return dataInconsistencies, nil
	})
}

func employeeCurrentMutateWithVersionDemo(config *Configuration, db *sql.DB, chOsSignal chan (os.Signal), employee *Employee) error {
	fmt.Println("\n==========================================")
	fmt.Println("--Testing Concurrent Mutate with Version--")
	fmt.Println("==========================================")
	return employeeConcurrentMutateDemo(config, chOsSignal, func(goRoutine, dataInconsistencies int) (int, error) {
		employeeRead, err := ReadEmployee(db, employee.EmailAddress)
		if err != nil {
			return dataInconsistencies, err
		}
		employeeUpdated, err := UpdateEmployeeWithVersion(db, employee, employeeRead.Version)
		if err != nil {
			return dataInconsistencies, err
		}
		if employeeUpdated.Version != employeeRead.Version+1 {
			dataInconsistencies++
		}
		return dataInconsistencies, nil
	})
}

func newMutex(config *Configuration) (interface {
	Mutex
	Close() error
}, error) {
	switch config.MutexType {
	default:
		return nil, errors.New("unsupported mutex type")
	case "redis_redshift":
		return NewRedSyncMutex(config)
	case "redis":
		mutex, err := NewRedisMutex(config)
		if err != nil {
			return nil, err
		}
		if err := mutex.Reset(); err != nil {
			return nil, err
		}
		return mutex, nil
	}
}

func Main(pwd string, args []string, envs map[string]string, chOsSignal chan os.Signal) error {
	const (
		firstName    string = "Antonio"
		lastName     string = "Alexander"
		emailAddress string = "antonio.alexander@mistersoftwaredeveloper.com"
	)

	config := ConfigFromEnv(envs)
	fmt.Printf("Configuration:\n mutex: %s\n go routines: %d\n duration: %s\n interval: %s\n",
		config.MutexType, config.GoRoutines, config.DemoDuration.String(), config.MutateInterval.String())
	db, err := NewSql(config)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("error occured while closing the database: \"%s\"\n", err)
		}
	}()
	mutex, err := newMutex(config)
	if err != nil {
		return err
	}
	defer func() {
		if err := mutex.Close(); err != nil {
			fmt.Printf("error occured while closing the mutex: \"%s\"\n", err)
		}
	}()
	if err := DeleteEmployee(db, emailAddress); err != nil {
		return err
	}
	employee, err := CreateEmployee(db, &Employee{
		FirstName:    firstName,
		LastName:     lastName,
		EmailAddress: emailAddress,
	})
	if err != nil {
		return err
	}
	if err := employeeCurrentMutateNoMutex(config, db, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithMutexDemo(config, db, mutex, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithMutexBenchmark(config, db, mutex, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithRowLockDemo(config, db, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithRowLockBenchmark(config, db, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithVersionDemo(config, db, chOsSignal, employee); err != nil {
		return err
	}
	if err := employeeCurrentMutateWithVersionBenchmark(config, db, chOsSignal, employee); err != nil {
		return err
	}
	return nil
}
