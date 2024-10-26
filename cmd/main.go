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

// Глобальные константы для настройки визуализации
const (
	screenWidth               = 800
	screenHeight              = 800
	bankRadius                = 50
	transactionAnimationSpeed = 0.01
	arrowThickness            = 5.0
	transactionSize           = 8
)

// Переменные для настройки цветов
var (
	arrowColor = color.Black
	textColor  = color.RGBA{B: 139, A: 255}
)

var (
	gameFont font.Face
)

func init() {
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

// Transaction представляет анимацию перевода средств между банками
type Transaction struct {
	FromX, FromY float64
	ToX, ToY     float64
	Amount       float64
	Progress     float64
	Color        color.RGBA
}

// BankSystem представляет основной объект алгоритма
type BankSystem struct {
	LambdaC     float64
	LambdaF     float64
	EnablePanic bool
	PanicRate   float64
	Banks       map[string]Bank
	game        *Game
}

// Bank представляет банк в банковской системе
type Bank struct {
	Balance      float64
	Dependencies map[string]float64
	Bankrupt     bool
	X, Y         float64
}

// Game представляет основной объект для визуализации
type Game struct {
	bankSystem    *BankSystem
	message       string
	nextStep      chan struct{}
	staticMessage string
	transactions  []Transaction
}

// Update это функция, которая обрабатывает обновления экрана
func (g *Game) Update() error {
	// Обновляем анимации транзакций
	for i := len(g.transactions) - 1; i >= 0; i-- {
		t := &g.transactions[i]
		t.Progress += transactionAnimationSpeed
		if t.Progress >= 1.0 {
			g.transactions = append(g.transactions[:i], g.transactions[i+1:]...)
		}
	}

	// Переход к следующему шагу визуализации
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		select {
		case g.nextStep <- struct{}{}:
		default:
		}
	}

	// Принудительный выход из программы
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}
	return nil
}

// drawText это функция для отрисовки текста на экране
func drawText(screen *ebiten.Image, str string, x, y int) {
	text.Draw(screen, str, gameFont, x, y, color.Black)
}

// drawBank это функция для отрисовки банка
func (g *Game) drawBank(screen *ebiten.Image, name string, bank Bank) {
	// Рисуем тень
	shadowColor := color.RGBA{A: 40}
	vector.DrawFilledCircle(screen, float32(bank.X+4), float32(bank.Y+4),
		bankRadius, shadowColor, true)

	// Определяем цвета для банка
	var bankFillColor, bankStrokeColor color.Color
	if bank.Bankrupt {
		// Для банкрота - бледно-красный фон и темно-красная обводка
		bankFillColor = color.RGBA{R: 255, G: 240, B: 240, A: 255}
		bankStrokeColor = color.RGBA{R: 180, A: 255}
	} else {
		// Для активного банка - белый фон и темно-зеленая обводка
		bankFillColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		bankStrokeColor = color.RGBA{G: 180, A: 255}
	}

	// Рисуем основной круг банка
	vector.DrawFilledCircle(screen, float32(bank.X), float32(bank.Y),
		bankRadius, bankFillColor, true)

	// Рисуем двойную обводку для эффекта глубины
	vector.StrokeCircle(screen, float32(bank.X), float32(bank.Y),
		bankRadius, 3, bankStrokeColor, true)
	vector.StrokeCircle(screen, float32(bank.X), float32(bank.Y),
		bankRadius-1, 1, bankStrokeColor, true)

	// Рисуем текст
	txt := fmt.Sprintf("%s\n%.1f", name, bank.Balance)
	drawText(screen, txt, int(bank.X)-15, int(bank.Y))
}

// addTransaction это функция для добавления транзакции с целью визуализации движения средств
func (g *Game) addTransaction(fromBank, toBank Bank, amount float64) {
	g.transactions = append(g.transactions, Transaction{
		FromX:    fromBank.X,
		FromY:    fromBank.Y,
		ToX:      toBank.X,
		ToY:      toBank.Y,
		Amount:   amount,
		Progress: 0,
		Color:    color.RGBA{G: 255, A: 255},
	})
}

// Draw это основная функция отрисовки
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	// Отображаем текст
	drawText(screen, g.message, 10, 20)
	drawText(screen, "Нажмите Enter для продолжения\n", 10, 40)
	drawText(screen, "Нажмите Esc для выхода\n", 10, screenHeight-10)
	drawText(screen, g.staticMessage, screenWidth-80, 20)

	// Рисуем стрелки
	for _, bank := range g.bankSystem.Banks {
		for debtor, amount := range bank.Dependencies {
			debtorBank := g.bankSystem.Banks[debtor]
			g.drawArrow(screen, bank.X, bank.Y, debtorBank.X, debtorBank.Y, amount)
		}
	}

	// Рисуем анимации транзакций
	for _, t := range g.transactions {
		currentX := t.FromX + (t.ToX-t.FromX)*t.Progress
		currentY := t.FromY + (t.ToY-t.FromY)*t.Progress

		// Рисуем частицу транзакции
		vector.DrawFilledCircle(screen, float32(currentX), float32(currentY),
			float32(transactionSize), t.Color, true)
	}

	// Рисуем банки поверх всего
	for name, bank := range g.bankSystem.Banks {
		g.drawBank(screen, name, bank)
	}
}

// Layout возвращает размеры экрана (является заглушкой для имплементации интерфейса ebiten.Game)
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// drawArrow это функция для отрисовки стрелки
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
			// Если суммы равны, рисуем одну изогнутую стрелку с двумя наконечниками
			startX := x1 + dx*bankRadius
			startY := y1 + dy*bankRadius
			endX := x2 - dx*bankRadius
			endY := y2 - dy*bankRadius

			// Рисуем основную линию с толщиной
			vector.StrokeLine(screen, float32(startX), float32(startY),
				float32(endX), float32(endY), arrowThickness, arrowColor, true)

			// Рисуем наконечники
			drawArrowHead(screen, endX, endY, dx, dy, arrowColor)
			drawArrowHead(screen, startX, startY, -dx, -dy, arrowColor)

			// Подпись значения с фоном
			midX := (startX + endX) / 2
			midY := (startY + endY) / 2
			txt := fmt.Sprintf("%.1f", bank1to2)
			drawTextWithBackground(screen, txt, int(midX), int(midY), textColor)
		} else {
			// Если суммы разные, рисуем две изогнутые параллельные линии
			offset := float64(15) // Уменьшенное расстояние между стрелками
			normalX := -dy * offset
			normalY := dx * offset

			// Первая линия
			startX1 := x1 + dx*bankRadius + normalX
			startY1 := y1 + dy*bankRadius + normalY
			endX1 := x2 - dx*bankRadius + normalX
			endY1 := y2 - dy*bankRadius + normalY

			// Вторая линия
			startX2 := x1 + dx*bankRadius - normalX
			startY2 := y1 + dy*bankRadius - normalY
			endX2 := x2 - dx*bankRadius - normalX
			endY2 := y2 - dy*bankRadius - normalY

			// Рисуем линии с толщиной
			vector.StrokeLine(screen, float32(startX1), float32(startY1),
				float32(endX1), float32(endY1), arrowThickness, arrowColor, true)
			vector.StrokeLine(screen, float32(startX2), float32(startY2),
				float32(endX2), float32(endY2), arrowThickness, arrowColor, true)

			// Рисуем наконечники
			drawArrowHead(screen, endX1, endY1, dx, dy, arrowColor)
			drawArrowHead(screen, endX2, endY2, -dx, -dy, arrowColor)

			// Подписи значений с фоном
			midX1 := (startX1 + endX1) / 2
			midY1 := (startY1 + endY1) / 2
			midX2 := (startX2 + endX2) / 2
			midY2 := (startY2 + endY2) / 2

			drawTextWithBackground(screen, fmt.Sprintf("%.1f", bank1to2),
				int(midX1), int(midY1), textColor)
			drawTextWithBackground(screen, fmt.Sprintf("%.1f", bank2to1),
				int(midX2), int(midY2), textColor)
		}
	} else {
		// Обычная однонаправленная стрелка
		startX := x1 + dx*bankRadius
		startY := y1 + dy*bankRadius
		endX := x2 - dx*bankRadius
		endY := y2 - dy*bankRadius

		// Рисуем линию с толщиной
		vector.StrokeLine(screen, float32(startX), float32(startY),
			float32(endX), float32(endY), arrowThickness, arrowColor, true)
		drawArrowHead(screen, endX, endY, dx, dy, arrowColor)

		// Подпись значения с фоном
		midX := (startX + endX) / 2
		midY := (startY + endY) / 2
		drawTextWithBackground(screen, fmt.Sprintf("%.1f", amount),
			int(midX), int(midY), textColor)
	}
}

// drawArrowHead это вспомогательная функция для рисования наконечника стрелки
func drawArrowHead(screen *ebiten.Image, x, y, dx, dy float64, color color.Color) {
	arrowSize := float64(12)
	angle := math.Pi / 4

	angle1 := math.Atan2(dy, dx) + angle
	angle2 := math.Atan2(dy, dx) - angle

	arrowX1 := x - arrowSize*math.Cos(angle1)
	arrowY1 := y - arrowSize*math.Sin(angle1)
	arrowX2 := x - arrowSize*math.Cos(angle2)
	arrowY2 := y - arrowSize*math.Sin(angle2)

	vector.StrokeLine(screen, float32(x), float32(y),
		float32(arrowX1), float32(arrowY1), arrowThickness/2, color, true)
	vector.StrokeLine(screen, float32(x), float32(y),
		float32(arrowX2), float32(arrowY2), arrowThickness/2, color, true)
}

// Вспомогательная функция для отрисовки текста с фоном
func drawTextWithBackground(screen *ebiten.Image, txt string, x, y int, textColor color.Color) {
	// Создаем белый фон с небольшой прозрачностью
	bgColor := color.RGBA{R: 255, G: 255, B: 255, A: 220}

	// Размеры текста для фона
	padding := 4
	width := len(txt)*7 + padding*2
	height := 15 + padding*2

	// Рисуем прямоугольник фона
	vector.DrawFilledRect(screen,
		float32(x-width/2), float32(y-height/2),
		float32(width), float32(height),
		bgColor, true)

	// Рисуем текст
	text.Draw(screen, txt, gameFont, x-width/2+padding, y+height/3, textColor)
}

// calculateBankPositions вычисляет координаты банков в системе
func calculateBankPositions(banks map[string]Bank) map[string]Bank {
	// Опять страшные математические приколы которые я что? Правильно, не буду объяснять, и так наобъяснялся сверху
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

// Bankruptcy основная функция для просчитывания последствий банкротства банков
func (s *BankSystem) Bankruptcy(bankruptBankName string) {
	// Очередь для обработки банкротств текущего уровня
	currentLevel := []string{bankruptBankName}
	s.game.message = fmt.Sprintf("Банк %s обанкротился", bankruptBankName)
	<-s.game.nextStep

	for len(currentLevel) > 0 {
		nextLevel := make([]string, 0)

		// Обрабатываем все банкротства текущего уровня
		for _, bankName := range currentLevel {
			bankruptBank := s.Banks[bankName]
			s.Banks[bankName] = bankruptBank

			// Запускаем панику для текущего банка
			s.BankRun(bankName)

			// Обрабатываем шок фондирования
			for partnerName, amount := range bankruptBank.Dependencies {
				partner := s.Banks[partnerName]
				if !partner.Bankrupt {
					shockImpact := amount * s.LambdaF
					partner.Balance -= shockImpact
					s.Banks[partnerName] = partner
					s.game.addTransaction(partner, bankruptBank, shockImpact)
					s.game.message = fmt.Sprintf("Шок фондирования в связи с банкротсвом банка %s: Банк %s потерял %.2f", bankName, partnerName, shockImpact)
					<-s.game.nextStep
				}
			}

			// Обрабатываем кредитный шок
			for partnerName, partner := range s.Banks {
				if creditAmount, exists := partner.Dependencies[bankName]; exists && !partner.Bankrupt {
					shockImpact := creditAmount * s.LambdaC
					partner.Balance -= shockImpact
					s.Banks[partnerName] = partner
					s.game.addTransaction(partner, bankruptBank, shockImpact)
					s.game.message = fmt.Sprintf("Кредитный шок в связи с банкротсвом банка %s: Банк %s потерял %.2f", bankName, partnerName, shockImpact)
					<-s.game.nextStep
				}
			}
		}

		// Проверяем новые банкротства для следующего уровня
		for bankName, bank := range s.Banks {
			if bank.Balance < 0 && !bank.Bankrupt {
				s.game.message = fmt.Sprintf("Банк %s обанкротился", bankName)
				<-s.game.nextStep
				nextLevel = append(nextLevel, bankName)
				bank.Bankrupt = true
				s.Banks[bankName] = bank
			}
		}

		// Переходим к следующему уровню
		currentLevel = nextLevel
	}
}

func (s *BankSystem) BankRun(bankruptBankName string) {
	if !s.EnablePanic {
		return
	}

	bankruptBank := s.Banks[bankruptBankName]
	partners := make(map[string]bool)

	// Кредиторы (те, кто вложил в банкрота)
	for bankName, bank := range s.Banks {
		if _, exists := bank.Dependencies[bankruptBankName]; exists {
			partners[bankName] = true
		}
	}

	// Должники (те, кому банкрот дал в долг)
	for debtor := range bankruptBank.Dependencies {
		partners[debtor] = true
	}

	// Симулируем набег на каждого партнера
	for partnerName := range partners {
		partner := s.Banks[partnerName]
		if !partner.Bankrupt { // Проверяем, что партнер еще не обанкротился

			// Закрываем долю p вкладов
			for bankName, bank := range s.Banks {
				if !bank.Bankrupt { // Проверяем что банк еще не обанкротился
					if amount, exists := bank.Dependencies[partnerName]; exists {
						partner.Balance -= amount * s.PanicRate
						s.Banks[partnerName] = partner
						bank.Balance += amount * s.PanicRate
						s.Banks[bankName] = bank

						s.game.addTransaction(partner, bank, amount*s.PanicRate)
						s.game.message = fmt.Sprintf("Набег вкладчиков: Банк %s забирает %.2f из своего вклада в банк %s в связи с банкротством банка %s",
							bankName, amount*s.PanicRate, partnerName, bankruptBankName)
						<-s.game.nextStep
					}
				}
			}
		}
	}
}

// StressTest функция для запуска стресс-теста
func (s *BankSystem) StressTest(bankName string) {
	s.game.message = "Начальное состояние банковской системы"
	<-s.game.nextStep
	s.game.message = fmt.Sprintf("Начало стресс-теста: банк %s объявляется банкротом", bankName)
	<-s.game.nextStep

	bank := s.Banks[bankName]
	bank.Bankrupt = true
	bank.Balance = -1
	s.Banks[bankName] = bank

	s.Bankruptcy(bankName)

	s.game.message = "Стресс-тест завершен"
	<-s.game.nextStep
}

func main() {
	X := 1000.0 // Баланс каждого банка
	Y := 5000.0 // Сумма задолженности каждого банка
	p := 0.7    // Процент от вклада который заберет банк при набеге
	lambda := 0.5

	banks := map[string]Bank{
		"1": {Balance: X, Dependencies: map[string]float64{"2": Y / 2, "5": Y / 2}},
		"2": {Balance: X, Dependencies: map[string]float64{"1": Y / 2, "3": Y / 2}},
		"3": {Balance: X, Dependencies: map[string]float64{"2": Y / 2, "4": Y / 2}},
		"4": {Balance: X, Dependencies: map[string]float64{"3": Y / 2, "5": Y / 2}},
		"5": {Balance: X, Dependencies: map[string]float64{"1": Y / 2, "4": Y / 2}},
	}

	banks = calculateBankPositions(banks)

	bankSystem := &BankSystem{
		LambdaC:     lambda,
		LambdaF:     lambda,
		Banks:       banks,
		PanicRate:   p,
		EnablePanic: true,
	}

	game := &Game{
		nextStep:      make(chan struct{}, 1),
		staticMessage: fmt.Sprintf("λc = %.2f\nλf = %.2f\np = %.2f\npanic = %t", bankSystem.LambdaC, bankSystem.LambdaF, bankSystem.PanicRate, bankSystem.EnablePanic),
		transactions:  make([]Transaction, 0),
	}

	bankSystem.game = game
	game.bankSystem = bankSystem

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Визуализация банковской системы")

	go bankSystem.StressTest("1")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
