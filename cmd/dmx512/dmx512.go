package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type gui struct {
	List   []string
	Serial serial.Port
	Values [513]byte
}

func (g *gui) UpdateList() {
	list, err := enumerator.GetDetailedPortsList()
	if err != nil {
		log.Panic(err)
	}
	g.List = g.List[:0]
	for _, item := range list {
		g.List = append(g.List, item.Name)
	}
}

func (g *gui) Select(name string) {
	if g.Serial != nil {
		g.Serial.Close()
	}
	port, err := serial.Open(name, &serial.Mode{
		BaudRate: 250000,
		DataBits: 8,
		StopBits: serial.TwoStopBits,
		Parity:   serial.NoParity,
	})
	if err != nil {
		log.Panic(err)
	}
	g.Serial = port
}

func (g *gui) Push(id int, value uint8) {
	if id > 0 && id < 513 {
		g.Values[id] = value
	} else {
		log.Print("Invalid channel id:", id)
	}
}

func runGui() {
	g := &gui{}
	g.UpdateList()
	myApp := app.New()

	// Create a new window
	myWindow := myApp.NewWindow("DMX512 Controller")
	myWindow.Resize(fyne.NewSize(800, 600))

	// Create content for the window
	// COM port selection
	comPortLabel := widget.NewLabel("Select COM Port:")
	comPortList := widget.NewSelect(g.List, func(selected string) {
		log.Print("Selected COM Port:", selected)
		g.Select(selected)
	})

	// Channel brightness controls
	brightnessLabel := widget.NewLabel("Channel Brightness:")
	brightnessGrid := container.NewGridWithColumns(6)

	for i := 1; i <= 32; i++ {
		channelLabel := widget.NewLabel(fmt.Sprintf("Channel %d:", i))
		brightnessSlider := widget.NewSlider(0, 255)
		brightnessSlider.OnChanged = func(value float64) {
			g.Push(i, byte(value))
			log.Printf("Channel %d brightness: %.0f\n", i, value)
		}

		brightnessGrid.Add(channelLabel)
		brightnessGrid.Add(brightnessSlider)
	}

	content := container.NewVBox(
		widget.NewLabel("DMX512 Controller"),
		comPortLabel,
		comPortList,
		brightnessLabel,
		brightnessGrid,
	)

	// Set the content of the window
	myWindow.SetContent(content)

	// Цикл отправки DMX-фреймов (нужно повторять ~25-44 Гц)
	go func() {
		for {
			sendDmxFrame(g.Serial, g.Values[:])
			// Пауза для соблюдения частоты обновления (ок. 30-40 кадров в секунду)
			time.Sleep(25 * time.Millisecond)
		}
	}()

	// Show and run the application
	myWindow.ShowAndRun()
}

func main() {
	serialPort := flag.String("com", "/dev/ttyUSB0", "COM порт для подключения")
	guiFlag := flag.Bool("gui", true, "Запустить GUI")
	flag.Parse()
	if *guiFlag {
		runGui()
		return
	}
	// Настройки для DMX512: 250000 baud, 8 data bits, 2 stop bits, no parity
	mode := &serial.Mode{
		BaudRate: 250000,
		DataBits: 8,
		StopBits: serial.TwoStopBits,
		Parity:   serial.NoParity,
	}

	// Открываем порт (замените на ваш, например COM3 или /dev/ttyUSB0)
	port, err := serial.Open(*serialPort, mode)
	if err != nil {
		log.Fatalf("Ошибка открытия порта: %v", err)
	}
	defer port.Close()

	// DMX Universe: 1 байт Start Code (0x00) + 512 каналов данных
	universe := make([]byte, 513)

	log.Println("FTDI DMX Контроллер готов.")
	log.Println("Команды: <канал>:<яркость> (например, 1:255). 'exit' для выхода.")

	// Цикл отправки DMX-фреймов (нужно повторять ~25-44 Гц)
	go func() {
		for {
			sendDmxFrame(port, universe)
			// Пауза для соблюдения частоты обновления (ок. 30-40 кадров в секунду)
			time.Sleep(25 * time.Millisecond)
		}
	}()

	// Чтение ввода пользователя
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "exit" {
			break
		}

		parts := strings.Split(text, ":")
		if len(parts) != 2 {
			log.Println("Неверный формат. Используйте канал:значение")
			continue
		}

		chanIdx, _ := strconv.Atoi(parts[0])
		value, _ := strconv.Atoi(parts[1])

		if chanIdx < 1 || chanIdx > 512 || value < 0 || value > 255 {
			log.Println("Ошибка: канал 1-512, значение 0-255")
			continue
		}

		universe[chanIdx] = byte(value)
		log.Printf("Канал %d -> %d\n", chanIdx, value)
	}
}

func sendDmxFrame(port serial.Port, data []byte) {
	if port == nil {
		return
	}
	// 1. Посылаем BREAK: принудительно устанавливаем линию в LOW
	port.Break(100 * time.Microsecond) // Минимум 88 мкс по стандарту DMX

	// 2. MAB (Mark After Break): линия должна быть в HIGH
	time.Sleep(12 * time.Microsecond) // Минимум 8 мкс по стандарту

	// 3. Отправка данных (Start code + 512 байт)
	_, err := port.Write(data)
	if err != nil {
		log.Printf("Ошибка записи: %v", err)
	}
}
