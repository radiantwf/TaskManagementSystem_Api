package dals

import (
	"TaskManagementSystem_Api/models/common"
	"TaskManagementSystem_Api/models/types"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"
)

// TaskDAL 定义
type TaskDAL struct {
	mongo *common.MongoSessionStruct
}

// GetTaskHeaders 定义
func (dal *TaskDAL) GetTaskHeaders(pageSize, pageNumber int) (taskGetList []*types.TaskHeader_Get, err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	dal.mongo.UseDB("local")
	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}

	if pageSize < 5 {
		pageSize = 5
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	var taskList []*types.TaskHeader
	err = dal.mongo.Collection.Find(nil).Sort("-id").Skip((pageNumber - 1) * pageSize).Limit(pageSize).All(&taskList)
	if err != nil {
		return
	}
	taskCount := len(taskList)
	taskGetList = make([]*types.TaskHeader_Get, taskCount, taskCount)
	for index, value := range taskList {
		taskGet := new(types.TaskHeader_Get)
		common.StructDeepCopy(value, taskGet)
		// 获取人员姓名
		emp := new(types.EmployeeName)
		if value.PrimaryExecutorObjectID != nil {
			err1 := dal.mongo.Db.C("M_Employees").FindId(*value.PrimaryExecutorObjectID).One(&emp)
			if err1 == nil {
				taskGet.PrimaryExecutor = emp.Name
			}
		}
		taskGetList[index] = taskGet
	}
	return
}

// GetTaskCount 定义
func (dal *TaskDAL) GetTaskCount() (counts map[string]int, err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	dal.mongo.UseDB("local")
	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}
	counts = make(map[string]int)
	// 所有任务数
	totalCount, err1 := dal.mongo.Collection.Find(nil).Count()
	if err1 != nil {
		err = err1
		return
	}
	counts["total"] = totalCount
	// 未开始任务数
	notStartedCount, err1 := dal.mongo.Collection.Find(bson.M{"status": "未开始"}).Count()
	if err1 != nil {
		err = err1
		return
	}
	counts["notStarted"] = notStartedCount
	// 进行中任务数
	onGoingCount, err1 := dal.mongo.Collection.Find(bson.M{"status": "进行中", "planningEndDate": bson.M{"$lte": time.Now()}}).Count()
	if err1 != nil {
		err = err1
		return
	}
	counts["onGoing"] = onGoingCount
	// 超时任务数
	overtimeCount, err1 := dal.mongo.Collection.Find(bson.M{"status": "进行中", "planningEndDate": bson.M{"$gt": time.Now()}}).Count()
	if err1 != nil {
		err = err1
		return
	}
	counts["overtime"] = overtimeCount
	return
}

// GetTaskDetail 定义
func (dal *TaskDAL) GetTaskDetail(id string) (taskGet *types.Task_Get, err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	dal.mongo.UseDB("local")
	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}
	taskGet = new(types.Task_Get)
	task := new(types.Task)
	dal.mongo.Collection.Find(bson.M{"id": id}).One(task)
	common.StructDeepCopy(task, taskGet)

	// 获取人员姓名
	emp := new(types.EmployeeName)
	err1 := dal.mongo.Db.C("M_Employees").FindId(task.PrimaryExecutorObjectID).One(&emp)
	if err1 == nil {
		taskGet.PrimaryExecutor = emp.Name
	}
	emp = new(types.EmployeeName)
	err1 = dal.mongo.Db.C("M_Employees").FindId(task.CreatorObjectID).One(&emp)
	if err1 == nil {
		taskGet.Creator = emp.Name
	}
	emp = new(types.EmployeeName)
	err1 = dal.mongo.Db.C("M_Employees").FindId(task.PrimarySellerObjectID).One(&emp)
	if err1 == nil {
		taskGet.PrimarySeller = emp.Name
	}
	emp = new(types.EmployeeName)
	err1 = dal.mongo.Db.C("M_Employees").FindId(task.PrimaryOCObjectID).One(&emp)
	if err1 == nil {
		taskGet.PrimaryOC = emp.Name
	}
	otherExecutorsCount := len(task.OtherExecutorObjectIds)
	taskGet.OtherExecutors = make([]string, otherExecutorsCount, otherExecutorsCount)
	for index, value := range task.OtherExecutorObjectIds {
		emp = new(types.EmployeeName)
		err1 = dal.mongo.Db.C("M_Employees").FindId(value).One(&emp)
		if err1 == nil {
			taskGet.OtherExecutors[index] = *emp.Name
		}
	}

	product := new(types.ProductName)
	err1 = dal.mongo.Db.C("T_Products").FindId(task.ParentProductObjectID).One(&emp)
	if err1 == nil {
		taskGet.ParentProduct = product.Name
	}
	project := new(types.ProjectName)
	err1 = dal.mongo.Db.C("T_Projects").FindId(task.ParentProjectObjectID).One(&emp)
	if err1 == nil {
		taskGet.ParentProject = project.Name
	}
	return
}

// AddTask 定义
func (dal *TaskDAL) AddTask(taskPost types.Task_Post) (err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	task := new(types.Task)
	common.StructDeepCopy(taskPost, task)
	task.OID = bson.NewObjectId()
	dal.mongo.UseDB("local")

	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}

	dateString := time.Now().Format("20060102")
	var maxID *types.MaxID
	taskRegex := "^T" + dateString + "[0-9]{4}$"
	iter := dal.mongo.Collection.Find(bson.M{"id": bson.M{"$regex": taskRegex}}).Sort("-id").Iter()
	iter.Next(&maxID)
	var id string
	if maxID == nil {
		id = "T" + dateString + "0001"
	} else {
		maxNum, _ := strconv.Atoi((*maxID.ID)[9:])
		if maxNum >= 9999 {
			err = errors.New("今天新建的任务超出最大限度")
			return
		}
		id = fmt.Sprintf("T%s%04d", dateString, maxNum+1)
	}
	task.ID = &id
	now := time.Now()
	task.CreatedTime = &now
	var status string
	if task.PrimaryOCID == nil {
		status = "新建"
	} else if task.PrimaryOCID != nil && task.PrimaryExecutorID == nil {
		status = "分配中"
	} else if task.PrimaryExecutorID != nil && task.PlanningBeginDate == nil {
		status = "计划中"
	} else if task.PlanningBeginDate != nil && task.PlanningEndDate != nil && task.RealBeginDate == nil && task.RealEndDate == nil {
		status = "未开始"
	}

	task.Status = &status
	percent := 0
	task.Percent = &percent
	objectID := new(types.ObjectID)
	err1 := dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": task.CreatorID}).One(&objectID)
	if err1 != nil || objectID.Oid == nil {
		task.CreatorID = nil
	} else {
		task.CreatorObjectID = objectID.Oid
	}
	objectID = new(types.ObjectID)
	dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": task.PrimarySellerID}).One(&objectID)
	if err1 != nil || objectID.Oid == nil {
		task.PrimarySellerID = nil
	} else {
		task.PrimarySellerObjectID = objectID.Oid
	}
	objectID = new(types.ObjectID)
	dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": task.PrimaryOCID}).One(&objectID)
	if err1 != nil || objectID.Oid == nil {
		task.PrimaryOCID = nil
	} else {
		task.PrimaryOCObjectID = objectID.Oid
	}
	objectID = new(types.ObjectID)
	dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": task.PrimaryExecutorID}).One(&objectID)
	if err1 != nil || objectID.Oid == nil {
		task.PrimaryExecutorID = nil
	} else {
		task.PrimaryExecutorObjectID = objectID.Oid
	}
	err = dal.mongo.Collection.Insert(task)
	if err != nil && strings.Contains(err.Error(), "E11000 duplicate key error collection:") {
		return dal.AddTask(taskPost)
	} else if err != nil {
		return
	}

	return
}

// DeleteTask 定义
func (dal *TaskDAL) DeleteTask(id string, user types.UserInfo_Get) (err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	dal.mongo.UseDB("local")
	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}

	task := new(types.Task)
	err = dal.mongo.Collection.Find(bson.M{"id": id}).One(task)
	if err != nil {
		return
	}
	if (*task.Status != "新建" && *task.Status != "未开始") && !user.CheckPermissions(1) {
		err = errors.New("当前任务已经开始，不能删除当前记录。")
		return
	}
	if *task.CreatorID != *user.EmpID && !user.CheckPermissions(1) {
		err = errors.New("与任务创建者不是同一用户，不能删除当前记录。")
		return
	}
	err = dal.mongo.Collection.RemoveId(task.OID)
	if err != nil {
		return
	}
	dal.mongo.Db.C("T_Communications").RemoveAll(bson.M{"relevantId": id})
	return
}

// UpdateTask 定义
func (dal *TaskDAL) UpdateTask(id string, task types.Task_Post, user types.UserInfo_Get) (err error) {
	dal.mongo, err = common.GetMongoSession()
	if err != nil {
		return
	}
	defer dal.mongo.CloseSession()
	dal.mongo.UseDB("local")
	err = dal.mongo.UseCollection("T_Tasks")
	if err != nil {
		return
	}
	m, err1 := dal.setUpdateBsonMap(task, user)
	if err1 != nil {
		err = err1
		return
	}

	err = dal.mongo.Collection.Update(bson.M{"id": id}, bson.M{"$set": m})
	if err != nil {
		return
	}

	return
}
func (dal *TaskDAL) setUpdateBsonMap(task types.Task_Post, user types.UserInfo_Get) (m map[string]interface{}, err error) {
	m = make(map[string]interface{})

	if task.Name != nil {
		m["name"] = *task.Name
	}
	if task.Resume != nil {
		m["resume"] = *task.Resume
	}
	if task.Description != nil {
		m["description"] = *task.Description
	}
	if task.CustomerContact != nil {
		m["customerContact"] = *task.CustomerContact
	}
	if task.CreatedTime != nil {
		m["createdTime"] = *task.CreatedTime
	}
	if task.PrimarySellerID != nil {
		objectID := new(types.ObjectID)
		err1 := dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": *task.PrimarySellerID}).One(&objectID)
		if err1 == nil && objectID.Oid != nil {
			m["primarySellerId"] = *task.PrimarySellerID
			m["PrimarySellerObjectID"] = *objectID.Oid
		} else {
			m["primarySellerId"] = nil
			m["PrimarySellerObjectID"] = nil
		}
	}
	if task.PrimaryOCID != nil {
		objectID := new(types.ObjectID)
		err1 := dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": *task.PrimaryOCID}).One(&objectID)
		if err1 == nil && objectID.Oid != nil {
			m["primaryOCId"] = *task.PrimaryOCID
			m["primaryOCObjectId"] = *objectID.Oid
		} else {
			m["primaryOCId"] = nil
			m["primaryOCObjectId"] = nil
		}
	}
	if task.PrimaryExecutorID != nil {
		objectID := new(types.ObjectID)
		err1 := dal.mongo.Db.C("M_Employees").Find(bson.M{"empId": *task.PrimaryExecutorID}).One(&objectID)
		if err1 == nil && objectID.Oid != nil {
			m["primaryExecutorId"] = *task.PrimaryExecutorID
			m["primaryExecutorObjectId"] = *objectID.Oid
		} else {
			m["primaryExecutorId"] = nil
			m["primaryExecutorObjectId"] = nil
		}
	}
	// OtherExecutorObjectIds  []bson.ObjectId `bson:"otherExecutorObjectIds"`
	// OtherExecutorIDs        []string        `bson:"otherExecutorIds"`

	if task.RequiringEndDate != nil {
		m["requiringEndDate"] = *task.RequiringEndDate
	}
	if task.PlanningBeginDate != nil {
		m["planningBeginDate"] = *task.PlanningBeginDate
	}
	if task.PlanningEndDate != nil {
		m["planningEndDate"] = *task.PlanningEndDate
	}
	if task.RealBeginDate != nil {
		m["realBeginDate"] = *task.RealBeginDate
	}
	if task.RealEndDate != nil {
		m["realEndDate"] = *task.RealEndDate
	}
	if task.Percent != nil {
		m["percent"] = *task.Percent
	}
	if task.Status != nil {
		m["status"] = *task.Status
	}
	if task.ParentProductID != nil {
		objectID := new(types.ObjectID)
		err1 := dal.mongo.Db.C("T_Products").Find(bson.M{"id": *task.ParentProductID}).One(&objectID)
		if err1 == nil && objectID.Oid != nil {
			m["parentProductId"] = *task.ParentProductID
			m["parentProductObjectId"] = *objectID.Oid
		} else {
			m["parentProductId"] = nil
			m["parentProductObjectId"] = nil
		}
	}
	if task.ParentProjectID != nil {
		objectID := new(types.ObjectID)
		err1 := dal.mongo.Db.C("T_Projects").Find(bson.M{"id": *task.ParentProjectID}).One(&objectID)
		if err1 == nil && objectID.Oid != nil {
			m["parentProjectId"] = *task.ParentProjectID
			m["parentProjectObjectId"] = *objectID.Oid
		} else {
			m["parentProjectId"] = nil
			m["parentProjectObjectId"] = nil
		}
	}
	if len(m) == 0 {
		err = errors.New("没有任何修改内容！")
		return
	}
	return m, err
}
