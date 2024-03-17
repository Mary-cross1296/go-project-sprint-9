package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var (
		//mu sync.Mutex
		i int64
	)
	i = 1
	for {
		select {
		case <-ctx.Done():
			close(ch) //Закрытие канала после отмены контекста
			return
		case ch <- i:
			fn(i)
			i++
		}
	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	var mu sync.Mutex
	for num := range in {
		mu.Lock()
		out <- num
		mu.Unlock()
		time.Sleep(1 * time.Millisecond)
	}
	close(out)
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		var mu sync.Mutex
		mu.Lock()
		inputSum += i
		inputCount++
		mu.Unlock()
	})

	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup
	//var mu sync.Mutex

	// 4. Собираем числа из каналов outs
	for i, out := range outs {
		wg.Add(1)
		go func(in <-chan int64, i int) {
			defer wg.Done()
			for v := range in {
				amounts[i]++
				chOut <- v
			}
		}(out, i)

		//go func(in <-chan int64, i int64) {
		//defer wg.Done()
		//var num int64
		//for {
		//select {
		//case num = <-in:
		//mu.Lock()
		//amounts[i]++
		//chOut <- num
		//mu.Unlock()
		//case _, ok := <-in:
		//if !ok {
		//return
		//}
		//}
		//}
		//}(out, int64(index))
	}

	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count int64 // количество чисел результирующего канала
	var sum int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	for numSum := range chOut {
		count++
		sum += numSum
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
