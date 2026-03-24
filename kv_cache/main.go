package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"kv_cache/kv"
)

type UserProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Company   string `json:"company"`
	JobTitle  string `json:"job_title"`
	CreatedAt int64  `json:"created_at"`
}

func main() {
	opts := kv.DefaultOptions
	db, err := kv.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	totalOps := 100000 // 10万并发量
	var wg sync.WaitGroup

	var writeSuccess int32
	var writeFailed int32

	fmt.Printf("阶段 1：开始 %d 万并发真实 JSON 写入测试\n", totalOps/10000)

	startWrite := time.Now()

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 核心优化：为每个协程创建一个无锁的 Faker 实例，避免 10 万协程抢夺全局随机数锁
			faker := gofakeit.NewUnlocked(int64(id) + time.Now().UnixNano())

			// 1. 瞬间生成高度逼真的变长数据
			user := UserProfile{
				ID:        fmt.Sprintf("user_%d", id),
				Name:      faker.Name(),
				Email:     faker.Email(),
				Phone:     faker.Phone(),
				Address:   faker.Address().Address,
				Company:   faker.Company(),
				JobTitle:  faker.JobTitle(),
				CreatedAt: time.Now().UnixNano(),
			}

			// 2. 将数据序列化为真实的 JSON 字节流
			valBytes, err := json.Marshal(user)
			if err != nil {
				atomic.AddInt32(&writeFailed, 1)
				return
			}

			key := []byte(user.ID)

			// 3. 极速落盘
			if err := db.Put(key, valBytes); err != nil {
				atomic.AddInt32(&writeFailed, 1)
			} else {
				atomic.AddInt32(&writeSuccess, 1)
			}
		}(i)
	}

	wg.Wait()
	_ = db.Sync() // 确保内存缓冲里的数据全部刷入物理硬盘

	writeCost := time.Since(startWrite)
	fmt.Printf("写入完成！成功: %d 条, 失败: %d 条\n", writeSuccess, writeFailed)
	fmt.Printf("写入总耗时: %v\n", writeCost)
	fmt.Printf("包含高强度 JSON 序列化的写入 TPS: %.0f 次/秒\n\n", float64(totalOps)/writeCost.Seconds())

	var readSuccess int32
	var readFailed int32

	fmt.Printf("阶段 2：开始 %d 万并发读取与 JSON 反序列化验证\n", totalOps/10000)

	startRead := time.Now()

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			expectedID := fmt.Sprintf("user_%d", id)
			key := []byte(expectedID)

			// 1. 无锁化极速读取磁盘二进制流
			valBytes, err := db.Get(key)
			if err != nil {
				atomic.AddInt32(&readFailed, 1)
				return
			}

			// 2. 反序列化回 Go 结构体
			var storedUser UserProfile
			if err := json.Unmarshal(valBytes, &storedUser); err != nil {
				atomic.AddInt32(&readFailed, 1)
				return
			}

			if storedUser.ID == expectedID && storedUser.Name != "" {
				atomic.AddInt32(&readSuccess, 1)
			} else {
				atomic.AddInt32(&readFailed, 1)
			}
		}(i)
	}

	wg.Wait()

	readCost := time.Since(startRead)
	fmt.Printf("读取与校验完成！成功比对: %d 条, 失败: %d 条\n", readSuccess, readFailed)
	fmt.Printf("读取总耗时: %v\n", readCost)
	fmt.Printf("包含 JSON 反序列化的读取 QPS: %.0f 次/秒\n\n", float64(totalOps)/readCost.Seconds())

	fmt.Println("结束！")
}
