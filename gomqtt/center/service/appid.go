package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"im.zgl/talents"
	"github.com/uber-go/zap"
)

type Return struct {
	Result  string        `json:"result"`
	Message string        `json:"msg"`
	Data    []interface{} `json:"data"`
}

func appidManage(c echo.Context) error {
	act := c.FormValue("action")
	var err error

	switch act {
	case "create_appid":
		err = createAppid(c)

	case "delete_appid":
		err = deleteAppid(c)

	case "update_appid":
		err = updateAppid(c)

	case "add_topics":
		err = addTopics(c)

	default:
		err = errors.New("action invalid")
	}

	if err != nil {
		Logger.Info("appid action error", zap.String("act", act), zap.Error(err))
		returnError(err.Error(), c)
	}

	return nil
}
func createAppid(c echo.Context) error {
	appid, user, pw, err := getAUP(c)
	if err != nil {
		return err
	}

	desc := c.FormValue("desc")
	compress, err := strconv.Atoi(c.FormValue("compress"))
	if err != nil {
		compress = 211
	}

	plType, err := strconv.Atoi(c.FormValue("payload_type"))
	if err != nil {
		return fmt.Errorf("payload_type invalid,err : %v, type: %v", err, c.FormValue("payload_type"))
	}

	var query string
	// 查询要插入数据是否已经存在
	query = fmt.Sprintf(`SELECT * FROM appid WHERE appid='%s'`, appid)
	res, err := db.Query(query)
	if err != nil {
		return err
	}

	if res.Next() {
		return errors.New("appid already exist")
	}

	// 不存在，插入数据

	query = fmt.Sprintf("INSERT INTO appid (`appid`, `payloadType`,`compress`,`user`,`password`,`desc` ,`inputDate`) VALUES ('%s','%d','%d','%s','%s','%s','%s')", appid, plType, compress, user, pw, desc, talents.Time2String(time.Now()))
	_, err = db.Exec(query)
	if err != nil {
		Logger.Info("insert sql error", zap.String("sql", query))
		return err
	}

	r := &Return{
		Result:  "success",
		Message: "create appid successful",
	}
	c.JSON(200, r)
	return nil
}

func updateAppid(c echo.Context) error {
	appid, user, pw, err := getAUP(c)
	if err != nil {
		return err
	}

	compress, err := strconv.Atoi(c.FormValue("compress"))
	if err != nil {
		return errors.New("invalid compress")
	}

	var query string
	// 查询要插入数据是否已经存在
	query = fmt.Sprintf("SELECT * FROM appid WHERE `appid`='%s' and `user`='%s' and `password`='%s'", appid, user, pw)
	res, err := db.Query(query)
	if err != nil {
		return err
	}

	if !res.Next() {
		return errors.New("appid not exist")
	}

	// 存在，更新数据
	query = fmt.Sprintf("UPDATE appid SET `compress`='%d',updateDate='%s' WHERE `appid`='%s'", compress, talents.Time2String(time.Now()), appid)
	_, err = db.Exec(query)
	if err != nil {
		Logger.Info("update appid error", zap.String("sql", query))
		return err
	}

	r := &Return{
		Result:  "success",
		Message: "update appid successful",
	}
	c.JSON(200, r)
	return nil
}

func deleteAppid(c echo.Context) error {
	appid, user, pw, err := getAUP(c)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM appid WHERE `appid`='%s' and `user`='%s' and `password`='%s'", appid, user, pw)
	res, err := db.Exec(query)
	if err != nil {
		return err
	}

	rn, _ := res.RowsAffected()
	if rn == 0 {
		return errors.New("appid  not exist OR user、password error")
	}

	r := &Return{
		Result:  "success",
		Message: "delete appid sucessful",
	}
	c.JSON(200, r)

	return nil
}

type Topics struct {
	TS map[string]*Topic `json:"topics"`
}

type Topic struct {
	Type string `json:"type"`
}

func addTopics(c echo.Context) error {
	appid, user, pw, err := getAUP(c)
	if err != nil {
		return err
	}

	topics := strings.Split(c.FormValue("topics"), ",")

	query := fmt.Sprintf("SELECT topics FROM appid WHERE `appid`='%s' AND `user`='%s' AND password='%s'", appid, user, pw)
	res, err := db.Query(query)
	if err != nil {
		return err
	}
	if !res.Next() {
		return errors.New("serviceID  not exist OR owner、password error")
	}

	var resBuf []byte
	err = res.Scan(&resBuf)
	if err != nil {
		return fmt.Errorf("scan error: %v", err)
	}

	ts := &Topics{
		TS: make(map[string]*Topic),
	}

	if string(resBuf) != "" {
		err = json.Unmarshal(resBuf, &ts)
		if err != nil {
			return fmt.Errorf("json decode error: %v,data : %v", err, string(resBuf))
		}
	}

	for _, t := range topics {
		ts1 := strings.Split(t, "--")
		if len(ts1) != 2 {
			return errors.New("topic must be topic--type representation")
		}

		topic, tp := ts1[0], ts1[1]
		ts.TS[topic] = &Topic{
			Type: tp,
		}
	}

	b, err := json.Marshal(ts)
	if err != nil {
		return err
	}

	// 存在，更新数据
	query = fmt.Sprintf("UPDATE appid SET `topics`='%s',updateDate='%s' WHERE `appid`='%s'", string(b), talents.Time2String(time.Now()), appid)
	_, err = db.Exec(query)
	if err != nil {
		Logger.Info("add topics error", zap.String("sql", query))
		return err
	}

	r := &Return{
		Result:  "success",
		Message: "add topics successful",
	}
	c.JSON(200, r)
	return nil
}

func deleteTopics(c echo.Context) error {

	return nil
}

func getAUP(c echo.Context) (string, string, string, error) {
	appid := c.FormValue("appid")
	user := c.FormValue("user")
	pw := c.FormValue("password")

	if appid == "" || user == "" || pw == "" {
		return "", "", "", errors.New("user or appid or password is empty")
	}

	return appid, user, pw, nil
}

func returnError(em string, c echo.Context) {
	r := &Return{
		Result:  "error",
		Message: em,
	}
	c.JSON(http.StatusOK, r)
}
