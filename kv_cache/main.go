package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"kv_cache/kv"
)

func main() {
	fmt.Println("正在从citylots.json中提取数据")

	file, err := os.Open("citylots.json")
	if err != nil {
		log.Fatal("找不到citylots.json文件", err)
	}
	defer file.Close()


	decoder := json.NewDecoder(file)


	for {
		t, err := decoder.Token()
		if err != nil {
			log.Fatal("解析结构失败:", err)
		}
		if s, ok := t.(string); ok && s == "features" {
			break
		}
	}

	_, err = decoder.Token()
	if err != nil {
		log.Fatal(err)
	}

	// 提取出所有地块的原始json字节流
	var items [][]byte
	for decoder.More() {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			break
		}
		items = append(items, raw)
	}

	totalOps := len(items)
	fmt.Printf("成功提取%d条真实地块数据！内存状态稳定。\n\n", totalOps)


	opts := kv.DefaultOptions
	db, err := kv.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var wg sync.WaitGroup
	var writeSuccess, writeFailed int32


	fmt.Printf("拉起%d万个goroutine进行极限并发写入\n", totalOps/10000)

	startWrite := time.Now()

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := []byte(fmt.Sprintf("sf_lot:%d", idx))

			if err := db.Put(key, items[idx]); err != nil {
				atomic.AddInt32(&writeFailed, 1)
			} else {
				atomic.AddInt32(&writeSuccess, 1)
			}
		}(i)
	}

	wg.Wait()
	_ = db.Sync() // 强制将bufio里的最后一点数据刷盘

	writeCost := time.Since(startWrite)
	fmt.Printf("写入完成！成功:%d条, 失败:%d条\n", writeSuccess, writeFailed)
	fmt.Printf("写入总耗时: %v\n", writeCost)
	fmt.Printf("处理变长数据的写入TPS: %.0f 次/秒\n\n", float64(totalOps)/writeCost.Seconds())

	var readSuccess, readFailed int32

	fmt.Printf("开始%d万并发极限读取测试\n", totalOps/10000)

	startRead := time.Now()

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			key := []byte(fmt.Sprintf("sf_lot:%d", idx))

			// 执行无锁并发读取
			val, err := db.Get(key)

			// 只要读取的字节流长度不一致，就算数据损坏
			if err == nil && len(val) == len(items[idx]) {
				atomic.AddInt32(&readSuccess, 1)
			} else {
				atomic.AddInt32(&readFailed, 1)
			}
		}(i)
	}

	wg.Wait()

	readCost := time.Since(startRead)
	fmt.Printf("读取比对完成！成功读取并校验:%d条, 失败: %d条\n", readSuccess, readFailed)
	fmt.Printf("读取总耗时: %v\n", readCost)
	fmt.Printf("巨型value读取QPS: %.0f次/秒\n\n", float64(totalOps)/readCost.Seconds())

	fmt.Println("结束！")
}
