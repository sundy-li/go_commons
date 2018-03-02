package gomysql

import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql"
    "strconv"
    "strings"
    "errors"
)

type Model struct {
    db        *sql.DB
    tablename string
}

func (m *Model) SetTableName(name string)  {
    m.tablename = name
}

func NewDbMysql(host string, port int, user string, password string, dbName string,charset string) (*Model,error) {
    c := new(Model)
    dataSourceName := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=%s", user, password, host, port, dbName,charset)
    db, err := sql.Open("mysql", dataSourceName)
    err = db.Ping()
    if err != nil {
        return c, err
    }
    c.db = db
    return c, err
}

func (m *Model) Insert(param map[string]interface{}) (num int, err error) {
    if m.db == nil {
        return 0, errors.New("mysql not connect")
    }
    var keys []string
    var values []string
    for key, value := range param {
        keys = append(keys, key)
        switch value.(type) {
        case int, int64, int32:
            values = append(values, strconv.Itoa(value.(int)))
        case string:
            values = append(values, value.(string))
        case float32, float64:
            values = append(values, strconv.FormatFloat(value.(float64), 'f', -1, 64))

        }

    }
    fileValue := "'" + strings.Join(values, "','") + "'"
    fileds := "`" + strings.Join(keys, "`,`") + "`"
    sql := fmt.Sprintf("INSERT INTO %v (%v) VALUES (%v)", m.tablename, fileds, fileValue)
    result, err := m.db.Exec(sql)
    if err != nil {
        defer func() {
            if err := recover(); err != nil {
                fmt.Printf("SQL syntax errors ")
            }
        }()
        return 0, err
    }
    i, err := result.LastInsertId()
    s, _ := strconv.Atoi(strconv.FormatInt(i, 10))
    return s, err

}

func (m *Model) InsertBatch(params []map[string]interface{}) (num int, err error) {
    if m.db == nil {
        return 0, errors.New("mysql not connect")
    }
    var keys []string
    paramOne := params[0]
    for key, _ := range paramOne {
        keys = append(keys, key)
    }
    var fileValues string
    for _, param := range params {
        var values []string
        for _,key := range keys {
            value,ok := param[key]
            if ok != true {
                return 0, errors.New("mysql key value dont match")
            }
            switch value.(type) {
            case int, int64, int32:
                values = append(values, strconv.Itoa(value.(int)))
            case string:
                values = append(values, value.(string))
            case float32, float64:
                values = append(values, strconv.FormatFloat(value.(float64), 'f', -1, 64))
            }
        }
        fileValue := "('" + strings.Join(values, "','") + "')"
        if fileValues == "" {
            fileValues = fileValues + fileValue
        }else {
            fileValues = fileValues + "," + fileValue
        }
    }
    fileds := "(`" + strings.Join(keys, "`,`") + "`)"
    sql := fmt.Sprintf("INSERT INTO %v %v VALUES %v", m.tablename, fileds, fileValues)
    result, err := m.db.Exec(sql)
    if err != nil {
        defer func() {
            if err := recover(); err != nil {
                fmt.Printf("SQL syntax errors ")
            }
        }()
        return 0, err
    }
    i, err := result.LastInsertId()
    s, _ := strconv.Atoi(strconv.FormatInt(i, 10))
    return s, err
}

func (m *Model) Close() {
    m.db.Close()
}