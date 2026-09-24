---
name: ECharts
tag: sb-echarts
category: data
summary: Apache ECharts driven by server-sent options, in the page's theme and language.
author: zweiundeins
tags: [chart, charts, echarts, graph, data, visualization]
since: 2026-09-23
preview: |
  <sb-echarts style="block-size: 9rem" option='{"grid":{"top":8,"bottom":20,"left":30,"right":8},"xAxis":{"type":"category","data":["Mon","Tue","Wed","Thu","Fri","Sat","Sun"]},"yAxis":{"type":"value"},"series":[{"type":"bar","data":[120,200,150,80,70,110,130]}]}'></sb-echarts>
usage: |
  <sb-echarts option='{"xAxis":{"type":"category","data":["Mon","Tue","Wed","Thu","Fri"]},"yAxis":{"type":"value"},"series":[{"type":"bar","data":[120,200,150,80,70]}]}'></sb-echarts>
playground:
  attrs: {option: '{"xAxis":{"type":"category","data":["Mercury","Venus","Earth","Mars"]},"yAxis":{"type":"value","name":"Moons"},"series":[{"type":"bar","data":[0,0,1,2]}]}'}
  exclude: [option]
  style: "block-size: 16rem"
---

Draws an [Apache ECharts](https://echarts.apache.org) chart from an option your server sends as JSON. The element does the part every page would otherwise write again: it colours the chart from the page's `--sb-*` tokens and follows theme changes, writes numbers, months and weekdays in the reader's language, keeps a wrapping legend clear of the axes, and animates each new option into place. ECharts itself is vendored with the component and loaded only when a chart comes near the screen.

Size it with CSS (`block-size`, `height`); it defaults to `18rem`.

## Examples

### From the server

Send any ECharts option. A string that is exactly `var(--token)` becomes that colour, so the server can colour a series with the page's own palette.

```html preview
<sb-echarts option='{
  "tooltip": {"trigger": "axis"},
  "legend": {"data": ["Launches", "Landings"], "top": 0},
  "xAxis": {"type": "category", "data": ["2022", "2023", "2024", "2025", "2026"]},
  "yAxis": {"type": "value"},
  "series": [
    {"name": "Launches", "type": "bar", "data": [61, 96, 134, 165, 190]},
    {"name": "Landings", "type": "line", "smooth": true, "data": [55, 90, 127, 159, 186], "itemStyle": {"color": "var(--sb-ok)"}}
  ]
}'></sb-echarts>
```

### Live

When the `option` attribute changes (a server morph, or a signal as here), the chart moves to the new state instead of redrawing.

```html preview
<div data-signals="{_fuel: [82, 64, 91, 48, 73]}">
  <sb-echarts style="block-size: 14rem" data-preserve-attr="option"
    data-attr:option="JSON.stringify({xAxis: {type: 'category', data: ['Tank A', 'Tank B', 'Tank C', 'Tank D', 'Tank E']}, yAxis: {type: 'value', max: 100, name: '%'}, series: [{type: 'bar', data: $_fuel}]})"></sb-echarts>
  <sb-button data-on:click="$_fuel = $_fuel.map(() => Math.round(20 + Math.random() * 80))">Refuel</sb-button>
</div>
```

### Beyond axes

Pies, gauges and the like get no phantom grid.

```html preview
<sb-echarts style="block-size: 16rem" option='{
  "tooltip": {"trigger": "item"},
  "series": [{"type": "pie", "radius": ["45%", "70%"], "data": [
    {"name": "Hydrogen", "value": 74}, {"name": "Helium", "value": 24}, {"name": "Everything else", "value": 2}
  ]}]
}'></sb-echarts>
```

### A fallback table

Put the same data inside the element. It shows until the chart has drawn (and without JavaScript), and afterwards it stays for screen readers, who get the numbers rather than a picture of them.

```html preview
<sb-echarts style="block-size: 12rem" option='{"xAxis":{"type":"category","data":["Q1","Q2","Q3","Q4"]},"yAxis":{"type":"value"},"series":[{"type":"line","data":[12,19,15,23]}]}'>
  <table>
    <caption>Signals received per quarter</caption>
    <tr><th>Q1</th><th>Q2</th><th>Q3</th><th>Q4</th></tr>
    <tr><td>12</td><td>19</td><td>15</td><td>23</td></tr>
  </table>
</sb-echarts>
```

### Clicks

A click on a data item fires `sb-chart-click`.

```html preview
<div data-signals="{_picked: ''}">
  <sb-echarts style="block-size: 12rem" data-on:sb-chart-click="$_picked = evt.detail.name + ': ' + evt.detail.value"
    option='{"xAxis":{"type":"category","data":["Io","Europa","Ganymede","Callisto"]},"yAxis":{"type":"value","name":"km"},"series":[{"type":"bar","data":[3643,3122,5268,4821]}]}'></sb-echarts>
  <p data-text="$_picked || 'Click a bar.'"></p>
</div>
```

## Charts whose shape is code

A custom series draws through a `renderItem` function, which no server can send as JSON. Define the chart once in the page's JavaScript and have the server name it:

```js
import { defineChartKind } from '/c/echarts/echarts.js'

defineChartKind('timeline', (option, { echarts, color, css, formatNumber, lang, width, height }) => ({
  xAxis: { type: 'time' },
  yAxis: { type: 'category', data: option.rows },
  series: [{ type: 'custom', data: option.bars, renderItem: (params, api) => { /* … */ } }],
}))
```

```html
<sb-echarts option='{"kind": "timeline", "rows": ["Apollo 11"], "bars": [[0, -14182940000, -13402140000]]}'></sb-echarts>
```

The builder gets the ECharts module (for `echarts.graphic` and friends), `color()` to resolve a token or CSS colour, `css()` to read a token such as a font, the element's number format, language and size. What it returns is themed like any other option.

## Numbers

Value axes and axis tooltips format numbers with `Intl.NumberFormat` in the element's language, unless the option sets a formatter of its own. If your server formats numbers differently (another separator, say), set the same rule for every chart on the page so the two never disagree:

```js
import { setNumberFormat } from '/c/echarts/echarts.js'
setNumberFormat((n, lang) => myFormat(n, lang))
```

## Theming

Colours come from the `--sb-*` tokens: text, borders and the tooltip surface from the semantic ones, series from `--sb-chart-1` to `--sb-chart-8` (falling back to brand, accent, ok, warn, danger, info…). Set those on the element or an ancestor to give charts their own palette. A pick on `sb-theme-switch`, or the system flipping under "auto", redraws the chart in the new colours without starting it again.

## Accessibility

With a fallback table in the element, screen readers get the table and the drawing is hidden from them. Without one, ECharts' own description is switched on, which names the chart type and series. With `prefers-reduced-motion: reduce` nothing animates.

## Licence

ECharts is © The Apache Software Foundation, under the Apache License 2.0; it is vendored unmodified from the `echarts@6.1.0` npm release (`vendor.json`), with its licence and notice in `vendor/`.
