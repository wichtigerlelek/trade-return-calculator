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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func generateDataset(startBalance float64, numTrades int, numSims int, riskPct float64, rr float64, baseWr float64, maxRiskAmt float64, slippage float64) [][]float64 {
	rand.Seed(time.Now().UnixNano())
	dataset := make([][]float64, numSims)

	for s := 0; s < numSims; s++ {
		dataset[s] = runSimulation(startBalance, numTrades, riskPct, rr, baseWr, maxRiskAmt, slippage)
	}
	return dataset
}

func runSimulation(startBalance float64, numTrades int, riskPct float64, rr float64, baseWr float64, maxRiskAmt float64, slippage float64) []float64 {
	history := make([]float64, numTrades+1)
	currentBalance := startBalance
	history[0] = currentBalance
	currentWR := baseWr

	for i := 1; i <= numTrades; i++ {
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
	if riskAmount > maxRiskAmt && maxRiskAmt != 0 {
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

type Analytics struct {
	MedianEndingBalance float64
	AverageMaxDrawdown  float64
	WorstMaxDrawdown    float64
	AverageSortino      float64
	TotalNetProfit      float64
}

func analyzeDataset(dataset [][]float64, startBalance float64) Analytics {
	numSims := len(dataset)
	numTrades := len(dataset[0]) - 1

	endingBalances := make([]float64, numSims)
	totalMaxDD := 0.0
	worstMaxDD := 0.0
	totalNetProfit := 0.0
	totalSortino := 0.0

	for s := 0; s < numSims; s++ {
		peak := dataset[s][0]
		simMaxDD := 0.0

		sumReturns := 0.0
		sumDownsideSq := 0.0

		for t := 1; t <= numTrades; t++ {
			val := dataset[s][t]
			prev := dataset[s][t-1]

			if val > peak {
				peak = val
			}
			if peak > 0 {
				dd := (peak - val) / peak
				if dd > simMaxDD {
					simMaxDD = dd
				}
			}

			ret := 0.0
			if prev > 0 {
				ret = (val - prev) / prev
			}
			sumReturns += ret

			if ret < 0 {
				sumDownsideSq += ret * ret
			}
		}

		endingBalances[s] = dataset[s][numTrades]
		totalNetProfit += (endingBalances[s] - startBalance)

		totalMaxDD += simMaxDD
		if simMaxDD > worstMaxDD {
			worstMaxDD = simMaxDD
		}

		avgReturn := sumReturns / float64(numTrades)
		downsideDev := math.Sqrt(sumDownsideSq / float64(numTrades))

		simSortino := 0.0
		if downsideDev > 0 {
			simSortino = avgReturn / downsideDev
		}
		totalSortino += simSortino
	}

	sort.Float64s(endingBalances)
	medianBalance := endingBalances[numSims/2]

	return Analytics{
		MedianEndingBalance: medianBalance,
		AverageMaxDrawdown:  totalMaxDD / float64(numSims),
		WorstMaxDrawdown:    worstMaxDD,
		AverageSortino:      totalSortino / float64(numSims),
		TotalNetProfit:      totalNetProfit,
	}
}

func generateLineItems(data []float64) []opts.LineData {
	items := make([]opts.LineData, len(data))
	for i, v := range data {
		items[i] = opts.LineData{Value: v}
	}
	return items
}

func monteCarlo(dataset [][]float64, startBalance float64) *charts.Line {
	numSims := len(dataset)
	numTrades := len(dataset[0]) - 1

	xAxis := make([]int, numTrades+1)
	for i := 0; i <= numTrades; i++ {
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
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "bottom", Type: "scroll"}),
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
		charts.WithYAxisOpts(opts.YAxis{Name: "Balance", Type: "log"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Trades"}),
	)

	line.SetXAxis(xAxis)

	for s := 0; s < numSims; s++ {
		line.AddSeries(fmt.Sprintf("Sim %d", s+1), generateLineItems(dataset[s]),
			charts.WithLineChartOpts(opts.LineChart{Symbol: "none"}),
		)
	}

	line.SetSeriesOptions(
		charts.WithMarkLineNameYAxisItemOpts(opts.MarkLineNameYAxisItem{Name: "Start Balance", YAxis: startBalance}),
		charts.WithMarkLineStyleOpts(opts.MarkLineStyle{Symbol: []string{"none", "none"}}),
	)
	return line
}

func percentile(dataset [][]float64, startBalance float64, risk float64, wr float64) *charts.Line {
	numSims := len(dataset)
	numTrades := len(dataset[0]) - 1

	p10, p50, p90 := make([]opts.LineData, numTrades+1), make([]opts.LineData, numTrades+1), make([]opts.LineData, numTrades+1)
	xAxis := make([]int, numTrades+1)

	for t := 0; t <= numTrades; t++ {
		xAxis[t] = t
		column := make([]float64, numSims)
		for s := 0; s < numSims; s++ {
			column[s] = dataset[s][t]
		}
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
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true), Top: "bottom", Type: "scroll"}),
		charts.WithYAxisOpts(opts.YAxis{Type: "log", Name: "Balance"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
	)

	line.SetXAxis(xAxis).
		AddSeries("90th Percentile", p90, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"})).
		AddSeries("Median (Target)", p50, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"})).
		AddSeries("10th Percentile", p10, charts.WithLineChartOpts(opts.LineChart{Symbol: "none"}))

	line.SetSeriesOptions(
		charts.WithAreaStyleOpts(opts.AreaStyle{Opacity: opts.Float(0.2)}),
		charts.WithMarkLineNameYAxisItemOpts(opts.MarkLineNameYAxisItem{Name: "Start Balance", YAxis: startBalance}),
		charts.WithMarkLineStyleOpts(opts.MarkLineStyle{Symbol: []string{"none", "none"}}),
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
			{label: "Trades per year", default_: "200"},
			{label: "Simulations", default_: "20"},
			{label: "Risk Per Trade", default_: "0.02"},
			{label: "Risk-Reward", default_: "1.0"},
			{label: "Win Rate", default_: "0.55"},
			{label: "Max $ Risk Cap", default_: "50000"},
			{label: "Slippage", default_: "0.005"},
		},
	}
}

func (m model) Init() tea.Cmd { return nil }

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
		if strings.Contains(f.label, "Balance") || strings.Contains(f.label, "Cap") {
			prefix = "$"
		} else if strings.Contains(f.label, "Reward") {
			prefix = "1:"
		}

		finalValue := prefix + display
		label := f.label + ":"

		if i == m.cursor {
			paddedLabel := fmt.Sprintf("%-22s", label)
			sb.WriteString(fmt.Sprintf("%s\033[1m%s\033[0m \033[36m%s\033[0m\n", cursor, paddedLabel, finalValue))
		} else {
			sb.WriteString(fmt.Sprintf("%s%-22s %s\n", cursor, label, finalValue))
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
	tradesPerYear := m.getInt(2, 200)
	numSimulations := m.getInt(3, 20)
	typicalRisk := m.getFloat(4, 0.05)
	riskReward := m.getFloat(5, 1.0)
	winRate := m.getFloat(6, 0.58)
	maxRisk := m.getFloat(7, 50000) // Set to 0 for infinite
	slip := m.getFloat(8, 0.005)

	dataset := generateDataset(startBalance, numberOfTrades, numSimulations, typicalRisk, riskReward, winRate, maxRisk, slip)
	analytics := analyzeDataset(dataset, startBalance)

	fmt.Println("\n=== Configuration ===")
	fmt.Printf("Start Balance:      $%.2f\n", startBalance)
	fmt.Printf("Number of Trades:   %d\n", numberOfTrades)
	fmt.Printf("Trades / year:      %d\n", tradesPerYear)
	fmt.Printf("Simulations:        %d\n", numSimulations)
	fmt.Printf("Typical Risk:       %.2f\n", typicalRisk)
	fmt.Printf("Risk-Reward:        1:%.1f\n", riskReward)
	fmt.Printf("Win Rate:           %.4f\n", winRate)
	fmt.Printf("Max Risk:           $%.2f\n", maxRisk)
	fmt.Printf("Slippage:           %.3f\n\n", slip)

	fmt.Println("=== Analytics ===")
	fmt.Printf("Break-Even WR:      %.4f\n", breakEvenWinRate(typicalRisk, riskReward))
	fmt.Printf("Median Ending:      $%.2f\n", analytics.MedianEndingBalance)
	fmt.Printf("Total Net P/L:      $%.2f\n", analytics.TotalNetProfit)
	fmt.Printf("Avg Max Drawdown:   %.2f%%\n", analytics.AverageMaxDrawdown*100)
	fmt.Printf("Worst Drawdown:     %.2f%%\n", analytics.WorstMaxDrawdown*100)
	fmt.Printf("Per-Trade Sortino:  %.4f\n", analytics.AverageSortino)
	fmt.Printf("Annualized Sortino: %.4f\n", analytics.AverageSortino*math.Sqrt(float64(tradesPerYear)))

	percentileChart := percentile(dataset, startBalance, typicalRisk, winRate)
	monteCarloChart := monteCarlo(dataset, startBalance)

	page := components.NewPage()
	page.PageTitle = "Trade Simulator"
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(percentileChart, monteCarloChart)

	f, _ := os.Create("chart.html")
	page.Render(f)
	f.Close()

	openBrowser("chart.html")
}
