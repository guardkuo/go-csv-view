package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const (
	ldioFilePath = "ldio.csv"
	LBALabel     = "LBA"
	DepthLabel   = "Depth(LD)"
)

func main() {
	http.HandleFunc("/", renderChartHandler)
	fmt.Println("伺服器啟動於 http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func renderChartHandler(w http.ResponseWriter, r *http.Request) {
	records, err := loadCsv(ldioFilePath)
	if err != nil {
		http.Error(w, "無法讀取 CSV: "+err.Error(), 500)
		return
	}

	xData, yData := prepareLDIOData(records)
	line := plotChart(xData, yData)

	line.Render(w)
}

func plotChart(xData []string, yData map[string][]opts.LineData) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: "shine", Width: "100%", Height: "800px",
		}),
		charts.WithTitleOpts(opts.Title{Title: "LD IO View | Infortrend (v2.2.4)"}),
		charts.WithDataZoomOpts(
			opts.DataZoom{Type: "slider", Start: 0, End: 2},
			opts.DataZoom{Type: "inside"},
		),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "LBA(GB)", Type: "value", Scale: opts.Bool(true)}),
	)

	line.ExtendYAxis(opts.YAxis{Name: "Depth", Type: "value", Scale: opts.Bool(true)})
	line.SetXAxis(xData)

	// --- 1. LBA 系列 (在 v2.2.4 中透過 LineChart 控制不畫線) ---
	line.AddSeries(LBALabel, yData[LBALabel],
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(false),
		}),
	)

	// 關鍵修正：使用 SetSeriesOptions 並透過匿名函數強行修改底層屬性
	// 這是 v2.2.4 避開編譯歧義最有效的方法
	line.SetSeriesOptions(
		func(s *charts.SingleSeries) {
			if s.Name == LBALabel {
				s.Type = "scatter" // 強制轉散點，線會消失
				s.SymbolSize = 2
				s.Symbol = "circle"
			}
			if s.Name == DepthLabel {
				s.Symbol = "none" // Depth 不顯示點
			}
		},
	)

	// --- 2. Depth 系列 ---
	line.AddSeries(DepthLabel, yData[DepthLabel],
		charts.WithLineChartOpts(opts.LineChart{
			YAxisIndex: 1, // 使用第二條 Y 軸
			Smooth:     opts.Bool(false),
		}),
	)

	return line
}

func prepareLDIOData(records [][]string) ([]string, map[string][]opts.LineData) {
	startRow := 0
	if len(records) > 0 && strings.Contains(records[0][0], "LD") {
		startRow = 1
	}

	x := make([]string, 0)
	y := make(map[string][]opts.LineData)
	var startTestTime uint64

	for _, record := range records[startRow:] {
		if len(record) < 7 {
			continue
		}

		startTVal, _ := strconv.ParseUint(record[1], 10, 64)
		if startTestTime == 0 {
			startTestTime = startTVal
			startTVal = 0
		} else {
			startTVal = startTVal - startTestTime
		}

		lbaVal, _ := strconv.ParseUint(record[4], 10, 64)
		depthVal, _ := strconv.ParseUint(record[6], 10, 64)

		x = append(x, strconv.FormatUint(startTVal, 10))
		y[LBALabel] = append(y[LBALabel], opts.LineData{Value: lbaVal >> 21})
		y[DepthLabel] = append(y[DepthLabel], opts.LineData{Value: depthVal})
	}
	return x, y
}

func loadCsv(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	return reader.ReadAll()
}
