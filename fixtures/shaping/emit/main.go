// Command emit prints one large, noisy tool output per kind for suites/shaping;
// the fact each task asks for is computed, so reading this file does not give it.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// mix is a fixed integer hash: the same output on every run and platform.
func mix(i int) uint32 {
	x := uint32(i)*2654435761 + 0x9e3779b9
	x ^= x >> 15
	x *= 2246822519
	x ^= x >> 13
	return x
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: emit test|log|json|table|terminal|small")
		os.Exit(2)
	}
	var b strings.Builder
	code := 0
	switch os.Args[1] {
	case "test":
		code = test(&b)
	case "log":
		logs(&b)
	case "json":
		jsonOut(&b)
	case "table":
		table(&b)
	case "terminal":
		terminal(&b)
	case "small":
		small(&b)
	default:
		fmt.Fprintln(os.Stderr, "emit: unknown kind", os.Args[1])
		os.Exit(2)
	}
	os.Stdout.WriteString(b.String())
	os.Exit(code)
}

const cases = 1500

func failing() int { return 700 + int(mix(7)%400) }

func test(b *strings.Builder) int {
	bad := failing()
	b.WriteString("=== RUN   TestLedger\n")
	for i := 1; i <= cases; i++ {
		fmt.Fprintf(b, "=== RUN   TestLedger/case_%03d\n", i)
		if i == bad {
			got := 1000 + int(mix(i)%900)
			want := got - 45
			fmt.Fprintf(b, "    ledger_test.go:57: balance = %d, want %d\n", got, want)
		}
	}
	b.WriteString("--- FAIL: TestLedger (0.31s)\n")
	for i := 1; i <= cases; i++ {
		if i == bad {
			fmt.Fprintf(b, "    --- FAIL: TestLedger/case_%03d (0.00s)\n", i)
		} else {
			fmt.Fprintf(b, "    --- PASS: TestLedger/case_%03d (0.00s)\n", i)
		}
	}
	b.WriteString("FAIL\nFAIL\texample.com/ledger\t0.342s\nFAIL\n")
	return 1
}

func logs(b *strings.Builder) {
	const lines = 1600
	bad := 700 + int(mix(11)%200)
	for i := range lines {
		ts := fmt.Sprintf("2026-10-19T08:%02d:%02dZ", i/60%60, i%60)
		if i == bad {
			fmt.Fprintf(b, "%s ERROR payment declined order=ord-%06x customer=c-%d reason=insufficient_funds\n", ts, mix(i)&0xffffff, 4000+i)
			continue
		}
		fmt.Fprintf(b, "%s INFO http request method=GET path=/api/items/%d status=200 duration_ms=%d\n", ts, mix(i)%5000, 3+mix(i)%40)
	}
}

var warehouses = []string{"bucharest-1", "cluj-2", "iasi-1", "timisoara-3", "brasov-2"}

func jsonOut(b *strings.Builder) {
	type item struct {
		SKU        string `json:"sku"`
		Warehouse  string `json:"warehouse"`
		Qty        int    `json:"qty"`
		PriceCents int    `json:"price_cents"`
	}
	items := make([]item, 1500)
	for i := range items {
		items[i] = item{
			SKU:        fmt.Sprintf("SKU-%05d", 30000+i*3),
			Warehouse:  warehouses[mix(i)%uint32(len(warehouses))],
			Qty:        int(mix(i+1000) % 500),
			PriceCents: 199 + int(mix(i+2000)%90000),
		}
	}
	out, _ := json.MarshalIndent(items, "", "  ")
	b.Write(out)
	b.WriteString("\n")
}

func table(b *strings.Builder) {
	regions := []string{"EU-North", "EU-South", "EU-West", "US-East", "US-West", "APAC"}
	b.WriteString("date,region,product,units,revenue_eur\n")
	for i := range 900 {
		day := 1 + i%28
		fmt.Fprintf(b, "2026-09-%02d,%s,P-%04d,%d,%d\n", day, regions[i%len(regions)], i, 1+mix(i)%300, 100+mix(i+5000)%99000)
	}
}

func terminal(b *strings.Builder) {
	const modules = 1500
	marked := 600 + int(mix(23)%300)
	for i := range modules {
		fmt.Fprintf(b, "  built module m%04d in %dms\n", i, 5+mix(i)%400)
		if i == marked {
			fmt.Fprintf(b, "  digest m%04d sha256=%08x%08x\n", i, mix(i+9000), mix(i+9001))
		}
	}
	fmt.Fprintf(b, "built %d modules\n", modules)
}

func small(b *strings.Builder) {
	b.WriteString("packing release bundle\n")
	for i := range 20 {
		fmt.Fprintf(b, "  added asset a%02d.bin (%d KiB)\n", i, 4+mix(i)%60)
	}
	fmt.Fprintf(b, "build id: hx-%04d-%02x\n", mix(31)%10000, mix(32)&0xff)
	b.WriteString("bundle written\n")
}
