/**
 * ApexCharts dark theme defaults for KloudDB Shield dashboards.
 * Charts are optional — CSS-based strategic visuals remain the default.
 */
export function apexDarkTheme() {
  return {
    chart: {
      background: 'transparent',
      foreColor: '#a8b0c4',
      toolbar: { show: false },
      fontFamily: 'inherit',
    },
    grid: { borderColor: '#424a5f' },
    theme: { mode: 'dark' },
    colors: ['#55a3d7', '#88bf57', '#e85d75', '#d4a84b'],
  };
}

export function mountApexChart(el, options) {
  if (typeof window.ApexCharts === 'undefined') {
    console.warn('ApexCharts not loaded');
    return null;
  }
  const chart = new window.ApexCharts(el, {
    ...apexDarkTheme(),
    ...options,
  });
  chart.render();
  return chart;
}

export function destroyChart(chart) {
  if (chart && typeof chart.destroy === 'function') chart.destroy();
}
