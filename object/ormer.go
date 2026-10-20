// Copyright 2026 The Caslingo Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"fmt"
	"time"

	"github.com/caslingo/caslingo/conf"
	_ "github.com/go-sql-driver/mysql" // db = mysql
	"github.com/xorm-io/xorm"
	"github.com/xorm-io/xorm/names"
	_ "modernc.org/sqlite" // db = sqlite
)

var ormer *Ormer

type Ormer struct {
	driverName     string
	dataSourceName string
	dbName         string
	Engine         *xorm.Engine
}

// InitAdapter connects to the database in app.conf, creating the database and tables if needed.
func InitAdapter() {
	var err error
	ormer, err = NewAdapter(conf.GetConfigString("driverName"), conf.GetConfigDataSourceName(), conf.GetConfigString("dbName"))
	if err != nil {
		panic(err)
	}

	err = ormer.createTable()
	if err != nil {
		panic(err)
	}
}

func NewAdapter(driverName string, dataSourceName string, dbName string) (*Ormer, error) {
	a := &Ormer{driverName: driverName, dataSourceName: dataSourceName, dbName: dbName}
	err := a.createDatabase()
	if err != nil {
		return nil, err
	}

	source := a.dataSourceName + a.dbName
	if a.driverName == "sqlite" {
		// one writer at a time: wait for the lock instead of failing with "database is locked"
		source = fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)", a.dbName)
	}
	a.Engine, err = xorm.NewEngine(a.driverName, source)
	if err != nil {
		return nil, err
	}
	a.Engine.SetMapper(names.SnakeMapper{})
	a.Engine.SetConnMaxLifetime(5 * time.Minute)
	if a.driverName == "sqlite" {
		a.Engine.SetMaxOpenConns(1)
	}
	return a, nil
}

func (a *Ormer) createDatabase() error {
	if a.driverName != "mysql" {
		return nil
	}

	engine, err := xorm.NewEngine(a.driverName, a.dataSourceName)
	if err != nil {
		return err
	}
	defer engine.Close()

	_, err = engine.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", a.dbName))
	return err
}

func (a *Ormer) createTable() error {
	tables := []interface{}{
		new(Project),
		new(Component),
		new(Unit),
		new(Translation),
		new(Suggestion),
		new(Change),
		new(Term),
	}
	for _, table := range tables {
		err := a.Engine.Sync2(table)
		if err != nil {
			return err
		}
	}
	return nil
}
