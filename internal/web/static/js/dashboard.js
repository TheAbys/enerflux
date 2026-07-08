const key = 'grid.consumption.power'
fetch("/api/v1/metrics/?key=" + key)
    .then(r => r.json())
    .then(data => {
        const ctx = document.getElementById('energyChart');

        const labels = data.map(item => item.TS);
        const values = data.map(item => item.Value);

        new Chart(ctx, {
            type: 'line',
            data: {
                labels: data.map(x =>
                    new Date(x.TS).toLocaleTimeString()
                ),
                datasets: [{
                    label: key,
                    data: values,
                    borderColor: "#245e2c",
                    backgroundColor: "#245e2c"
                }]
            }
        })
    });