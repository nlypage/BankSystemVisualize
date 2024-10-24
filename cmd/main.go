package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 800
	screenHeight = 800
	bankRadius   = 50
)

var (
	gameFont font.Face
)

func init() {
	// Загружаем шрифт
	fontData, err := os.ReadFile("C:\\Windows\\Fonts\\arial.ttf")
	if err != nil {
		log.Fatal(err)
	}

	tt, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatal(err)
	}

	gameFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    13,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type BankSystem struct {
	LambdaC float64
	LambdaF float64
	Banks   map[string]Bank
	game    *Game
}

type Bank struct {
	Balance      float64
	Dependencies map[string]float64
	Bankrupt     bool
	X, Y         float64 // Координаты банка на экране
}

type Game struct {
	bankSystem *BankSystem
	message    string        // Сообщение события вверху экрана
	nextStep   chan struct{} // Канал для перехода к следующему шагу визуализации
}

func (g *Game) Update() error {
	// Обрабатываем нажатие Enter на клавиатуре для перехода к следующему шагу визуализации
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		select {
		case g.nextStep <- struct{}{}:
		default:
		}
	}
	return nil
}

// drawText рисует текст черным цветом на экране
func drawText(screen *ebiten.Image, str string, x, y int) {
	text.Draw(screen, str, gameFont, x, y, color.Black)
}

// Draw рисует каждый новый кадр визуализации банковской системы на экран
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	// Отображаем текущее событие вверху экрана
	drawText(screen, g.message, 10, 20)
	drawText(screen, "Нажмите Enter для продолжения", 10, 40)

	// Сначала рисуем все стрелки
	for _, bank := range g.bankSystem.Banks {
		for debtor, amount := range bank.Dependencies {
			debtorBank := g.bankSystem.Banks[debtor]
			g.drawArrow(screen, bank.X, bank.Y, debtorBank.X, debtorBank.Y, amount)
		}
	}

	// Затем рисуем банки поверх стрелок
	for name, bank := range g.bankSystem.Banks {
		bankColor := color.RGBA{G: 255, A: 255}
		if bank.Bankrupt {
			bankColor = color.RGBA{R: 255, A: 255}
		}

		vector.DrawFilledCircle(screen, float32(bank.X), float32(bank.Y), bankRadius, bankColor, true)
		vector.StrokeCircle(screen, float32(bank.X), float32(bank.Y), bankRadius, 2, color.Black, true)

		text := fmt.Sprintf("%s\n%.1f", name, bank.Balance)
		drawText(screen, text, int(bank.X)-10, int(bank.Y)+5)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawArrow рисует стрелку на экране отображающую зависимость между двумя банками
func (g *Game) drawArrow(screen *ebiten.Image, x1, y1, x2, y2, amount float64) {
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)

	dx /= length
	dy /= length

	// Проверяем, есть ли обратная зависимость
	bank1to2 := amount
	bank2to1 := float64(0)

	// Находим обратную зависимость
	for name, bank := range g.bankSystem.Banks {
		if bank.X == x1 && bank.Y == y1 {
			for targetName, targetBank := range g.bankSystem.Banks {
				if targetBank.X == x2 && targetBank.Y == y2 {
					if dependencyAmount, exists := bank.Dependencies[targetName]; exists {
						bank1to2 = dependencyAmount
					}
					if dependencyAmount, exists := targetBank.Dependencies[name]; exists {
						bank2to1 = dependencyAmount
					}
					break
				}
			}
			break
		}
	}

	// Если есть зависимость в обе стороны
	if bank2to1 > 0 {
		if bank1to2 == bank2to1 {
			// Если суммы равны, рисуем одну стрелку с двумя наконечниками
			startX := x1 + dx*bankRadius
			startY := y1 + dy*bankRadius
			endX := x2 - dx*bankRadius
			endY := y2 - dy*bankRadius

			vector.StrokeLine(screen, float32(startX), float32(startY), float32(endX), float32(endY), 1, color.Black, true)

			// Рисуем наконечники с обеих сторон
			drawArrowHead(screen, endX, endY, dx, dy)
			drawArrowHead(screen, startX, startY, -dx, -dy)

			// Подпись значения посередине
			midX := (startX + endX) / 2
			midY := (startY + endY) / 2
			drawText(screen, fmt.Sprintf("%.1f", bank1to2), int(midX), int(midY))
		} else {
			// Если суммы разные, рисуем две параллельные линии
			offset := float64(8000) // расстояние между двумя параллельными стрелками
			normalX := -dy * offset / length
			normalY := dx * offset / length

			// Первая линия (сдвинутая влево)
			startX1 := x1 + dx*bankRadius + normalX
			startY1 := y1 + dy*bankRadius + normalY
			endX1 := x2 - dx*bankRadius + normalX
			endY1 := y2 - dy*bankRadius + normalY

			// Вторая линия (сдвинутая вправо)
			startX2 := x1 + dx*bankRadius - normalX
			startY2 := y1 + dy*bankRadius - normalY
			endX2 := x2 - dx*bankRadius - normalX
			endY2 := y2 - dy*bankRadius - normalY

			// Рисуем первую стрелку
			vector.StrokeLine(screen, float32(startX1), float32(startY1), float32(endX1), float32(endY1), 1, color.Black, true)
			drawArrowHead(screen, endX1, endY1, dx, dy)

			// Рисуем вторую стрелку
			vector.StrokeLine(screen, float32(startX2), float32(startY2), float32(endX2), float32(endY2), 1, color.Black, true)
			drawArrowHead(screen, endX2, endY2, -dx, -dy)

			// Подписи значений
			midX1 := (startX1 + endX1) / 2
			midY1 := (startY1 + endY1) / 2
			midX2 := (startX2 + endX2) / 2
			midY2 := (startY2 + endY2) / 2

			drawText(screen, fmt.Sprintf("%.1f", bank1to2), int(midX1), int(midY1))
			drawText(screen, fmt.Sprintf("%.1f", bank2to1), int(midX2), int(midY2))
		}
	} else {
		// Обычная однонаправленная стрелка
		startX := x1 + dx*bankRadius
		startY := y1 + dy*bankRadius
		endX := x2 - dx*bankRadius
		endY := y2 - dy*bankRadius

		vector.StrokeLine(screen, float32(startX), float32(startY), float32(endX), float32(endY), 1, color.Black, true)
		drawArrowHead(screen, endX, endY, dx, dy)

		midX := (startX + endX) / 2
		midY := (startY + endY) / 2
		drawText(screen, fmt.Sprintf("%.1f", amount), int(midX), int(midY))
	}
}

// Вспомогательная функция для рисования наконечника стрелки
func drawArrowHead(screen *ebiten.Image, x, y, dx, dy float64) {
	arrowSize := float64(10)
	angle := math.Pi / 6

	angle1 := math.Atan2(dy, dx) + angle
	angle2 := math.Atan2(dy, dx) - angle

	arrowX1 := x - arrowSize*math.Cos(angle1)
	arrowY1 := y - arrowSize*math.Sin(angle1)
	arrowX2 := x - arrowSize*math.Cos(angle2)
	arrowY2 := y - arrowSize*math.Sin(angle2)

	vector.StrokeLine(screen, float32(x), float32(y), float32(arrowX1), float32(arrowY1), 1, color.Black, true)
	vector.StrokeLine(screen, float32(x), float32(y), float32(arrowX2), float32(arrowY2), 1, color.Black, true)
}

// calculateBankPositions вычисляет координаты банков в системе
func calculateBankPositions(banks map[string]Bank) map[string]Bank {
	// Опять страшные математические приколы которые я что? Правильно, не буду объяснять
	numBanks := len(banks)
	angle := 2 * math.Pi / float64(numBanks)
	centerX := float64(screenWidth) / 2
	centerY := float64(screenHeight) / 2
	radius := float64(screenHeight) / 3

	i := 0
	for name, bank := range banks {
		bank.X = centerX + radius*math.Cos(float64(i)*angle)
		bank.Y = centerY + radius*math.Sin(float64(i)*angle)
		banks[name] = bank
		i++
	}
	return banks
}

// Bankruptcy обработка банкротства каждого отдельного банка
func (s *BankSystem) Bankruptcy(bankruptBankName string) {
	bankruptBank := s.Banks[bankruptBankName]
	bankruptBank.Bankrupt = true
	s.Banks[bankruptBankName] = bankruptBank
	s.game.message = fmt.Sprintf("Банк %s обанкротился", bankruptBankName)
	<-s.game.nextStep

	// Просчитываем шок фондирования
	for bankName, amount := range bankruptBank.Dependencies {
		bank := s.Banks[bankName]
		shockImpact := amount * s.LambdaF
		bank.Balance -= shockImpact
		s.Banks[bankName] = bank
		s.game.message = fmt.Sprintf("Шок фондирования: Банк %s потерял %.2f", bankName, shockImpact)
		<-s.game.nextStep

		if bank.Balance < 0 && !bank.Bankrupt {
			s.game.message = fmt.Sprintf("Банк %s обанкротился из-за шока фондирования", bankName)
			<-s.game.nextStep
			s.Bankruptcy(bankName)
		}
	}

	// Просчитываем кредитный шок
	for bankName, bank := range s.Banks {
		if creditAmount, exists := bank.Dependencies[bankruptBankName]; exists {
			shockImpact := creditAmount * s.LambdaC
			bank.Balance -= shockImpact
			s.Banks[bankName] = bank
			s.game.message = fmt.Sprintf("Кредитный шок: Банк %s потерял %.2f", bankName, shockImpact)
			<-s.game.nextStep

			if bank.Balance <= 0 && !bank.Bankrupt {
				s.game.message = fmt.Sprintf("Банк %s обанкротился из-за кредитного шока", bankName)
				<-s.game.nextStep
				s.Bankruptcy(bankName)
			}
		}
	}
}

func (s *BankSystem) StressTest(bankName string) {
	s.game.message = fmt.Sprintf("Начало стресс-теста: банк %s объявляется банкротом", bankName)
	<-s.game.nextStep

	bank := s.Banks[bankName]
	bank.Bankrupt = true
	bank.Balance = 0
	s.Banks[bankName] = bank

	s.Bankruptcy(bankName)

	s.game.message = "Стресс-тест завершен"
	<-s.game.nextStep
}

func main() {
	banks := map[string]Bank{
		"1": {Balance: 2000, Dependencies: map[string]float64{"2": 2000, "3": 2000}},
		"2": {Balance: 4000, Dependencies: map[string]float64{"1": 1000, "3": 5000}},
		"3": {Balance: 4000, Dependencies: map[string]float64{"2": 3000, "1": 1000}},
	}

	banks = calculateBankPositions(banks)

	game := &Game{
		nextStep: make(chan struct{}, 1),
		message:  "Начальное состояние банковской системы",
	}

	bankSystem := &BankSystem{
		LambdaC: 0.5,
		LambdaF: 1,
		Banks:   banks,
		game:    game,
	}
	game.bankSystem = bankSystem

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Визуализация банковской системы")

	go bankSystem.StressTest("1")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
