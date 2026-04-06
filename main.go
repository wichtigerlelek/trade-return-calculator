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

func monteCarlo(startBalance float64, numTrades int, numSims int, risk float64, rr float64, wr float64) *charts.Line {
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

		for i := 1; i < numTrades+1; i++ {
			tradeWon := rand.Float64() <= wr
			if tradeWon {
				currentBalance += currentBalance * risk * rr
			} else {
				currentBalance -= currentBalance * risk
			}
			yAxis[i] = currentBalance
		}

		line.AddSeries(fmt.Sprintf("Sim %d", sim+1), generateLineItems(yAxis),
			charts.WithLineChartOpts(opts.LineChart{
				Symbol: "none",
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 1,
			}),
		)
	}

	line.SetSeriesOptions(
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

func percentile(startBalance float64, numTrades int, numSims int, risk float64, rr float64, wr float64) *charts.Line {
	allResults := make([][]float64, numTrades+1)
	for i := range allResults {
		allResults[i] = make([]float64, numSims)
	}

	for s := 0; s < numSims; s++ {
		curr := startBalance
		allResults[0][s] = curr
		for t := 1; t <= numTrades; t++ {
			if rand.Float64() <= wr {
				curr *= (1 + risk*rr)
			} else {
				curr *= (1 - risk)
			}
			allResults[t][s] = curr
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
			{label: "Typical Risk", default_: "0.05"},
			{label: "Risk-Reward", default_: "1.0"},
			{label: "Win Rate", default_: "0.58"},
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
	sb.WriteString("╔══════════════════════════════════╗\n")
	sb.WriteString("║       Trade Simulator Setup      ║\n")
	sb.WriteString("╚══════════════════════════════════╝\n\n")
	sb.WriteString("  Use ↑↓ to navigate, Enter to confirm\n\n")

	for i, f := range m.fields {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}

		display := f.value
		if display == "" {
			display = f.default_
		}

		if i == m.cursor {
			sb.WriteString(fmt.Sprintf("%s\033[1m%-20s\033[0m \033[36m%s\033[0m\n", cursor, f.label+":", display))
		} else {
			sb.WriteString(fmt.Sprintf("%s%-20s %s\n", cursor, f.label+":", display))
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

	fmt.Println("\n=== Configuration ===")
	fmt.Printf("Start Balance:    %.2f\n", startBalance)
	fmt.Printf("Number of Trades: %d\n", numberOfTrades)
	fmt.Printf("Simulations:      %d\n", numSimulations)
	fmt.Printf("Typical Risk:     %.4f\n", typicalRisk)
	fmt.Printf("Risk-Reward:      1:%.1f\n", riskReward)
	fmt.Printf("Win Rate:         %.4f\n", winRate)
	fmt.Printf("Break-Even WR:    %.4f\n", breakEvenWinRate(typicalRisk, riskReward))

	percentileChart := percentile(startBalance, numberOfTrades, numSimulations, typicalRisk, riskReward, winRate)
	monteCarloChart := monteCarlo(startBalance, numberOfTrades, numSimulations, typicalRisk, riskReward, winRate)

	page := components.NewPage()
	page.PageTitle = "Trade Simulator"
	page.SetLayout(components.PageFlexLayout)
	page.AddCharts(percentileChart, monteCarloChart)

	f, _ := os.Create("chart.html")
	page.Render(f)
	f.Close()

	openBrowser("chart.html")
}
