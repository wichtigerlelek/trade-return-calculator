package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func generateLineItems(data []float64) []opts.LineData {
	items := make([]opts.LineData, len(data))
	for i, v := range data {
		items[i] = opts.LineData{Value: v}
	}
	return items
}

func runSimulation(startBalance float64, numTrades int, riskPct float64, rr float64, baseWr float64, maxRiskAmt float64, slippage float64) []float64 {
	history := make([]float64, numTrades+1)
	currentBalance := startBalance
	history[0] = currentBalance
	currentWR := baseWr

	for i := 1; i <= numTrades; i++ {
		// Every 50 trades, the WR "drifts" by +/- 2%
		if i%50 == 0 {
			drift := (rand.Float64() * 0.04) - 0.02
			currentWR = baseWr + drift
		}

		currentBalance = calculateTrade(currentBalance, riskPct, maxRiskAmt, rr, currentWR, slippage)

		if currentBalance < 0 {
			currentBalance = 0
		}

		history[i] = currentBalance
	}
	return history
}

func calculateTrade(balance float64, riskPct float64, maxRiskAmt float64, rr float64, wr float64, slippage float64) float64 {
	riskAmount := balance * riskPct
	if riskAmount > maxRiskAmt {
		riskAmount = maxRiskAmt
	}

	tradeWon := rand.Float64() <= wr

	if tradeWon {
		balance += (riskAmount * (rr - slippage))
	} else {
		balance -= (riskAmount * (1 + slippage))
	}

	return balance
}

func monteCarlo(startBalance float64, numTrades int, numSims int, risk float64, rr float64, baseWr float64, maxRisk float64, slippage float64) *charts.Line {
	xAxis := make([]int, numTrades+1)
	for i := 0; i < numTrades+1; i++ {
		xAxis[i] = i
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "1200px", Height: "700px"}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Monte Carlo Lines",
			Subtitle: fmt.Sprintf("%d Trades | %d Scenarios", numTrades, numSims),
			Left:     "center",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "bottom",
			Type: "scroll",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
			Formatter: opts.FuncOpts(`
        function(params) {
            params.sort(function(a, b) {
                return b.value - a.value;
            });
            var result = params[0].axisValueLabel + '<br/>';
            params.forEach(function(item) {
                result += item.marker + item.seriesName + ': ' + item.value.toFixed(2) + '<br/>';
            });
            return result;
			}
			`),
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Balance",
			Type: "log",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Trades",
		}),
	)

	line.SetXAxis(xAxis)

	for sim := 0; sim < numSims; sim++ {
		currentBalance := startBalance
		yAxis := make([]float64, numTrades+1)
		yAxis[0] = currentBalance

		currentWR := baseWr

		for i := 1; i < numTrades+1; i++ {
			// Every 50 trades, the WR "drifts" by +/- 2%
			if i%50 == 0 {
				drift := (rand.Float64() * 0.04) - 0.02
				currentWR = baseWr + drift
			}

			currentBalance = calculateTrade(currentBalance, risk, maxRisk, rr, currentWR, slippage)

			if currentBalance <= 0 {
				currentBalance = 0
			}
			yAxis[i] = currentBalance
		}

		line.AddSeries(fmt.Sprintf("Sim %d", sim+1), generateLineItems(yAxis),
			charts.WithLineChartOpts(opts.LineChart{Symbol: "none"}),
		)
	}

	return line
}

func percentile(startBalance float64, numTrades int, numSims int, risk float64, rr float64, wr float64, maxRisk float64, slippage float64) *charts.Line {
	allResults := make([][]float64, numTrades+1)
	for i := range allResults {
		allResults[i] = make([]float64, numSims)
	}

	for s := 0; s < numSims; s++ {
		simHistory := runSimulation(startBalance, numTrades, risk, rr, wr, maxRisk, slippage)
		for t := 0; t <= numTrades; t++ {
			allResults[t][s] = simHistory[t]
		}
	}

	p10 := make([]opts.LineData, numTrades+1)
	p50 := make([]opts.LineData, numTrades+1)
	p90 := make([]opts.LineData, numTrades+1)
	xAxis := make([]int, numTrades+1)

	for t := 0; t <= numTrades; t++ {
		xAxis[t] = t
		column := allResults[t]
		sort.Float64s(column)
		p10[t] = opts.LineData{Value: column[int(float64(numSims)*0.1)]}
		p50[t] = opts.LineData{Value: column[int(float64(numSims)*0.5)]}
		p90[t] = opts.LineData{Value: column[int(float64(numSims)*0.9)]}
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "1200px", Height: "700px"}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Monte Carlo: Percentile Cloud",
			Subtitle: fmt.Sprintf("Risk: %.1f%% | Win Rate: %.2f%%", risk*100, wr*100),
			Left:     "center",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "bottom",
			Type: "scroll",
		}),
		charts.WithYAxisOpts(opts.YAxis{Type: "log", Name: "Balance"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
	)

	line.SetXAxis(xAxis).
		AddSeries("90th Percentile", p90, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"})).
		AddSeries("Median (Target)", p50, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"})).
		AddSeries("10th Percentile", p10, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"}))

	line.SetSeriesOptions(
		charts.WithAreaStyleOpts(opts.AreaStyle{
			Opacity: opts.Float(0.2),
		}),
		charts.WithMarkLineNameYAxisItemOpts(
			opts.MarkLineNameYAxisItem{
				Name:  "Start Balance",
				YAxis: startBalance,
			},
		),
		charts.WithMarkLineStyleOpts(opts.MarkLineStyle{
			Symbol: []string{"none", "none"},
		}),
	)

	return line
}

type field struct {
	label    string
	value    string
	default_ string
}

type model struct {
	fields []field
	cursor int
	done   bool
}

func initialModel() model {
	return model{
		fields: []field{
			{label: "Start Balance", default_: "10000"},
			{label: "Number of Trades", default_: "2000"},
			{label: "Simulations", default_: "20"},
			{label: "Risk Per Trade", default_: "0.02"},
			{label: "Risk-Reward", default_: "1.0"},
			{label: "Win Rate", default_: "0.55"},
			{label: "Max $ Risk Cap", default_: "50000"},
			{label: "Slippage", default_: "0.005"},
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.cursor == len(m.fields)-1 {
				m.done = true
				return m, tea.Quit
			}
			m.cursor++

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown:
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}

		case tea.KeyBackspace:
			f := &m.fields[m.cursor]
			if len(f.value) > 0 {
				f.value = f.value[:len(f.value)-1]
			}

		default:
			if msg.Type == tea.KeyRunes {
				m.fields[m.cursor].value += string(msg.Runes)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var sb strings.Builder
	sb.WriteString("╔═════════════════════════════════╗\n")
	sb.WriteString("║     TRADE RETURN CALCULATOR     ║\n")
	sb.WriteString("╚═════════════════════════════════╝\n\n")

	for i, f := range m.fields {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		display := f.value
		if display == "" {
			display = f.default_
		}

		prefix := ""
		// Use Contains instead of exact match to be safer with your labels
		if strings.Contains(f.label, "Balance") || strings.Contains(f.label, "Cap") {
			prefix = "$"
		} else if strings.Contains(f.label, "Reward") {
			prefix = "1:"
		}

		finalValue := prefix + display
		label := f.label + ":"

		if i == m.cursor {
			paddedLabel := fmt.Sprintf("%-22s", label)
			sb.WriteString(fmt.Sprintf("%s\033[1m%s\033[0m \033[36m%s\033[0m\n",
				cursor, paddedLabel, finalValue))
		} else {
			sb.WriteString(fmt.Sprintf("%s%-22s %s\n",
				cursor, label, finalValue))
		}
	}

	sb.WriteString("\n  \033[90mPress Enter on last field to run\033[0m\n")
	return sb.String()
}

func (m model) getFloat(i int, def float64) float64 {
	v := strings.TrimSpace(m.fields[i].value)
	if v == "" {
		return def
	}
	val, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return val
}

func (m model) getInt(i int, def int) int {
	v := strings.TrimSpace(m.fields[i].value)
	if v == "" {
		return def
	}
	val, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return val
}

func breakEvenWinRate(risk, riskReward float64) float64 {
	loss := math.Log(1 - risk)
	win := math.Log(1 + risk*riskReward)
	return loss / (loss - win)
}

func openBrowser(path string) {
	var cmd string
	var args []string

	absPath, _ := filepath.Abs(path)
	url := "file://" + absPath

	switch runtime.GOOS {
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	case "darwin":
		cmd, args = "open", []string{url}
	default:
		cmd, args = "xdg-open", []string{url}
	}

	exec.Command(cmd, args...).Start()
}

func main() {
	p := tea.NewProgram(initialModel())
	result, err := p.Run()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	m := result.(model)
	if !m.done {
		fmt.Println("Cancelled.")
		return
	}

	startBalance := m.getFloat(0, 10000.0)
	numberOfTrades := m.getInt(1, 2000)
	numSimulations := m.getInt(2, 20)
	typicalRisk := m.getFloat(3, 0.05)
	riskReward := m.getFloat(4, 1.0)
	winRate := m.getFloat(5, 0.58)
	maxRisk := m.getFloat(6, 50000) // The maximum amount of money per trade limited by liquidity
	slip := m.getFloat(7, 0.005)    // Difference between the expected price and the real price for a trade (influenced by spreads, volatility, volume, ...)

	fmt.Println("\n=== Configuration ===")
	fmt.Printf("Start Balance:    %.2f$\n", startBalance)
	fmt.Printf("Number of Trades: %d\n", numberOfTrades)
	fmt.Printf("Simulations:      %d\n", numSimulations)
	fmt.Printf("Typical Risk:     %.4f\n", typicalRisk)
	fmt.Printf("Risk-Reward:      1:%.1f\n", riskReward)
	fmt.Printf("Win Rate:         %.4f\n", winRate)
	fmt.Printf("Max Risk:         %.2f$\n", maxRisk)
	fmt.Printf("Slippage:         %.4f\n", slip)
	fmt.Printf("Break-Even WR:    %.4f\n", breakEvenWinRate(typicalRisk, riskReward))

	percentileChart := percentile(startBalance, numberOfTrades, numSimulations, typicalRisk, riskReward, winRate, maxRisk, slip)
	monteCarloChart := monteCarlo(startBalance, numberOfTrades, numSimulations, typicalRisk, riskReward, winRate, maxRisk, slip)

	page := components.NewPage()
	page.PageTitle = "Trade Simulator"
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(percentileChart, monteCarloChart)

	f, _ := os.Create("chart.html")
	page.Render(f)
	f.Close()

	openBrowser("chart.html")
}
