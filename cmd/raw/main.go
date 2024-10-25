package main

import (
	"fmt"
)

type BankSystem struct {
	LambdaC     float64 // Параметр lambda для кредитного шока
	LambdaF     float64 // Параметр lambda для шока фондирования
	EnablePanic bool    // Параметр для включения / выключения паники
	PanicRate   float64 // Параметр p для доли закрываемых вкладов

	Banks map[string]Bank
}

type Bank struct {
	Balance      float64
	Dependencies map[string]float64
	Bankrupt     bool
}

// Bankruptcy обработка банкротства каждого отдельного банка
func (s *BankSystem) Bankruptcy(bankruptBankName string) {
	// Очередь для обработки банкротств текущего уровня
	currentLevel := []string{bankruptBankName}

	for len(currentLevel) > 0 {
		nextLevel := make([]string, 0)

		// Обрабатываем все банкротства текущего уровня
		for _, bankName := range currentLevel {
			bankruptBank := s.Banks[bankName]
			bankruptBank.Bankrupt = true
			s.Banks[bankName] = bankruptBank

			// Запускаем панику для текущего банка
			s.BankRun(bankName)

			// Обрабатываем шок фондирования
			for partnerName, amount := range bankruptBank.Dependencies {
				bank := s.Banks[partnerName]
				if !bank.Bankrupt {
					shockImpact := amount * s.LambdaF
					bank.Balance -= shockImpact
					s.Banks[partnerName] = bank
				}
			}

			// Обрабатываем кредитный шок
			for partnerName, bank := range s.Banks {
				if creditAmount, exists := bank.Dependencies[bankName]; exists && !bank.Bankrupt {
					shockImpact := creditAmount * s.LambdaC
					bank.Balance -= shockImpact
					s.Banks[partnerName] = bank
				}
			}
		}

		// Проверяем новые банкротства для следующего уровня
		for bankName, bank := range s.Banks {
			if bank.Balance < 0 && !bank.Bankrupt {
				nextLevel = append(nextLevel, bankName)
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

	// Находим всех партнеров обанкротившегося банка
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
					}
				}
			}
		}
	}
}

func (s *BankSystem) StressTest(bankName string) int {
	bank := s.Banks[bankName]
	bank.Bankrupt = true
	bank.Balance = -1
	s.Banks[bankName] = bank
	s.Bankruptcy(bankName)

	bankruptedCount := 0
	for _, b := range s.Banks {
		if b.Bankrupt {
			bankruptedCount++
		}
	}
	return bankruptedCount
}

func main() {
	X := 1000.0  // Баланс каждого банка
	Y := 10000.0 // Сумма задолженности каждого банка

	banksFull := map[string]Bank{
		"1": {Balance: X, Dependencies: map[string]float64{"2": Y / 4, "3": Y / 4, "4": Y / 4, "5": Y / 4}},
		"2": {Balance: X, Dependencies: map[string]float64{"1": Y / 4, "3": Y / 4, "4": Y / 4, "5": Y / 4}},
		"3": {Balance: X, Dependencies: map[string]float64{"1": Y / 4, "2": Y / 4, "4": Y / 4, "5": Y / 4}},
		"4": {Balance: X, Dependencies: map[string]float64{"1": Y / 4, "2": Y / 4, "3": Y / 4, "5": Y / 4}},
		"5": {Balance: X, Dependencies: map[string]float64{"1": Y / 4, "2": Y / 4, "3": Y / 4, "4": Y / 4}},
	}

	banksCircle := map[string]Bank{
		"1": {Balance: X, Dependencies: map[string]float64{"2": Y / 2, "5": Y / 2}},
		"2": {Balance: X, Dependencies: map[string]float64{"1": Y / 2, "3": Y / 2}},
		"3": {Balance: X, Dependencies: map[string]float64{"2": Y / 2, "4": Y / 2}},
		"4": {Balance: X, Dependencies: map[string]float64{"3": Y / 2, "5": Y / 2}},
		"5": {Balance: X, Dependencies: map[string]float64{"1": Y / 2, "4": Y / 2}},
	}
	for p := 0.1; p <= 1.0; p += 0.1 {
		for lambda := 0.1; lambda <= 1.0; lambda += 0.1 {
			fullSystem := &BankSystem{
				LambdaC:     lambda,
				LambdaF:     lambda,
				Banks:       cloneBanks(banksFull),
				PanicRate:   p,
				EnablePanic: true,
			}

			circleSystem := &BankSystem{
				LambdaC:     lambda,
				LambdaF:     lambda,
				Banks:       cloneBanks(banksCircle),
				PanicRate:   p,
				EnablePanic: true,
			}

			fullCount := fullSystem.StressTest("1")
			circleCount := circleSystem.StressTest("1")

			if circleCount < fullCount {
				fmt.Printf("p: %f, lambda: %f\n", p, lambda)
			}
		}
	}
}

func cloneBanks(original map[string]Bank) map[string]Bank {
	clonedBanks := make(map[string]Bank)
	for k, v := range original {
		dependenciesCopy := make(map[string]float64)
		for dk, dv := range v.Dependencies {
			dependenciesCopy[dk] = dv
		}
		clonedBanks[k] = Bank{
			Balance:      v.Balance,
			Dependencies: dependenciesCopy,
			Bankrupt:     v.Bankrupt,
		}
	}
	return clonedBanks
}
