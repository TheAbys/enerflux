# TODO

## Laufende Wartung
- [ ] Dependabot-PRs wöchentlich prüfen und geeignete Updates übernehmen; die automatischen Prüfungen sind eingerichtet.
- [ ] PostgreSQL-Major-Upgrade von Version 16 mit Migrations- und Rollback-Plan vorbereiten.

## Rückfragen und offene Produktentscheidungen

### Betrieb und Datenversorgung
- [ ] Die beiden Helm-Releases auf dem Zielcluster deployen und persistenten Datenbankbetrieb einschließlich Neustart sowie Backup-/Restore prüfen. Dafür werden Clusterzugriff und ein abgestimmtes Wartungsfenster benötigt; siehe [k3s-Deployment-Anleitung](k3s/README.md).
- [ ] MQTT-Geräte reparieren. Vorher betroffene Geräte, Topics und Symptome erfassen.

### Preise und historische Daten
- [ ] Wasser- und Abwasserpreise mit Gültigkeit bis 31.12.2026 und ab 01.01.2027 pflegen. Die Berechnung kann nach Nennung beider Preise je Zeitraum umgesetzt werden.
- [ ] Alte Stromzählerstände importieren und Jahresverbrauch berechnen. Noch festzulegen sind Datenquelle, Zählerkennung, Einheit und Umgang mit Zählerwechseln.
- [ ] Pellet-Marktpreise optional importieren. Der DEPV veröffentlicht Monatswerte nach Liefermenge und Region; die offiziellen Seiten dokumentieren keine öffentliche API. Bitte Quelle, Region/PLZ, Liefermenge und Netto-/Bruttodarstellung festlegen, bevor eine externe Abhängigkeit eingebaut wird: [DEPV-Pelletpreis](https://www.depv.de/pelletpreis/) · [DEPI-Pelletpreis](https://www.depi.de/pelletpreis-wirtschaftlichkeit/).

### Stromtarife
- [ ] Tarifmodell festlegen: ein aktueller Tarif gegen ein Angebot oder Tarifhistorie je Zähler mit Angebotsvergleich.
- [ ] Festlegen, wie Grundgebühren, Einmalzahlungen und Neukundenboni über den Vergleichszeitraum verrechnet werden.
- [ ] Prüfen, ob die Tabelle „Letzte Messwerte“ für Strom ebenfalls Filter oder Gruppierung braucht. Im Graphen lassen sich die Strom-Messreihen jetzt einzeln ein- und ausblenden.
