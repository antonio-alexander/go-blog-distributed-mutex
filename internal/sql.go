package internal

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

const tableEmployee string = "employee"

func readEmployee(tx *sql.Tx, emailAddress string) (*Employee, error) {
	query := fmt.Sprintf("SELECT email_address, first_name, last_name, version FROM %s WHERE email_address=?;", tableEmployee)
	row := tx.QueryRow(query, emailAddress)
	if err := row.Err(); err != nil {
		return nil, err
	}
	employee := &Employee{}
	if err := row.Scan(
		&employee.EmailAddress,
		&employee.FirstName,
		&employee.LastName,
		&employee.Version,
	); err != nil {
		return nil, err
	}
	return employee, nil
}

func NewSql(config *Configuration) (*sql.DB, error) {
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=%t",
		config.MysqlUsername, config.MysqlPassword, config.MysqlHost,
		config.MysqlPort, config.MysqlDatabase, config.MysqlParseTime)
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func CreateEmployee(db *sql.DB, employee *Employee) (*Employee, error) {
	if employee == nil {
		return nil, errors.New("employee is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("INSERT INTO %s (email_address, first_name, last_name) VALUES (?, ?, ?);",
		tableEmployee)
	if _, err := tx.Exec(query,
		employee.EmailAddress, employee.FirstName, employee.LastName); err != nil {
		return nil, err
	}
	employee, err = readEmployee(tx, employee.EmailAddress)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return employee, nil
}

func ReadEmployee(db *sql.DB, emailAddress string) (*Employee, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	employee, err := readEmployee(tx, emailAddress)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return employee, nil
}

func UpdateEmployee(db *sql.DB, employee *Employee) (*Employee, error) {
	if employee == nil {
		return nil, errors.New("employee is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := fmt.Sprintf("UPDATE %s SET first_name = ?, last_name = ?, version = version+1 WHERE email_address=?;", tableEmployee)
	if _, err := tx.Exec(query,
		employee.FirstName, employee.LastName, employee.EmailAddress); err != nil {
		return nil, err
	}
	employee, err = readEmployee(tx, employee.EmailAddress)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return employee, nil
}

func UpdateEmployeeWithLock(db *sql.DB, employee *Employee) (*Employee, *Employee, error) {
	if employee == nil {
		return nil, nil, errors.New("employee is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	query := fmt.Sprintf("SELECT email_address, first_name, last_name, version FROM %s WHERE email_address = ? FOR UPDATE;", tableEmployee)
	row := tx.QueryRow(query, employee.EmailAddress)
	if err := row.Err(); err != nil {
		return nil, nil, err
	}
	employeeRead := &Employee{}
	if err := row.Scan(
		&employeeRead.EmailAddress,
		&employeeRead.FirstName,
		&employeeRead.LastName,
		&employeeRead.Version,
	); err != nil {
		return nil, nil, err
	}
	query = fmt.Sprintf("UPDATE %s SET first_name = ?, last_name = ?, version = version+1 WHERE email_address=?;", tableEmployee)
	if _, err := tx.Exec(query,
		employee.FirstName, employee.LastName, employee.EmailAddress); err != nil {
		return nil, nil, err
	}
	employeeUpdated, err := readEmployee(tx, employee.EmailAddress)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return employeeRead, employeeUpdated, nil
}

func UpdateEmployeeWithVersion(db *sql.DB, employee *Employee, version int) (*Employee, error) {
	if employee == nil {
		return nil, errors.New("employee is nil")
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	query := fmt.Sprintf("UPDATE %s SET first_name = ?, last_name = ?, version = version+1 WHERE email_address=? AND version=?;", tableEmployee)
	result, err := tx.Exec(query,
		employee.FirstName, employee.LastName, employee.EmailAddress, version)
	if err != nil {
		return nil, err
	}
	if n, err := result.RowsAffected(); err != nil {
		return nil, err
	} else if n <= 0 {
		return nil, errors.New("update failed; no rows affected")
	}
	employee, err = readEmployee(tx, employee.EmailAddress)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return employee, nil
}

func DeleteEmployee(db *sql.DB, emailAddress string) error {
	query := fmt.Sprintf("DELETE from %s WHERE email_address=?", tableEmployee)
	if _, err := db.Exec(query, emailAddress); err != nil {
		return err
	}
	return nil
}
