# Aggiornamenti e risoluzione dei problemi

Usa questa pagina quando le installazioni, gli aggiornamenti o gli avvii di Bedrock Nexus non si comportano come previsto.

## Problemi comuni

### I Gaming Services mancano o sono danneggiati

- Riparali o installali tramite il flusso guidato.
- Se necessario, rivedi lo stato di installazione di Minecraft Bedrock in Microsoft Store.

### GameInput manca

- Installa il pacchetto ridistribuibile richiesto quando Bedrock Nexus te lo chiede.
- Riavvia il launcher dopo l'installazione, se necessario.

### Il percorso di installazione non è scrivibile

- Sposta il percorso dei contenuti gestiti in una posizione scrivibile nelle **Impostazioni**.
- Se il launcher richiede l'elevazione dei privilegi per l'installazione o l'auto-aggiornamento, consentila quando ti fidi dell'azione.

### Bedrock Nexus si chiude subito o non mostra mai una finestra

- Su Windows, Bedrock Nexus ora avvia WebView2 in una modalità più sicura con rendering software per ridurre gli arresti anomali all'avvio su alcune combinazioni di GPU e driver.
- Se il launcher continua a fallire prima che appaia l'interfaccia, controlla `%APPDATA%\BedrockNexus.exe\logs\startup.log`.
- Riprova una volta dopo un avvio non riuscito. Se succede ancora, allega `startup.log` quando apri un problema su GitHub.

### Una versione non si avvia

- prova la stessa versione senza mod
- conferma che i componenti di Windows richiesti esistano ancora
- verifica che la versione abbia completato correttamente l'installazione
- riprova con una configurazione di prova isolata e pulita

### I download sono troppo lenti o falliscono ripetutamente

- cambia mirror
- riprova più tardi
- usa un'origine pacchetto locale, se disponibile

## Passaggi di ripristino sicuri

1. Fai il backup dei mondi importanti.
2. Rimuovi mod o pacchetti recenti.
3. Prova con una versione isolata pulita.
4. Reinstalla la versione interessata, se necessario.

## Comportamento dell'auto-aggiornamento

Bedrock Nexus può controllare, scaricare e installare gli aggiornamenti dell'app. Alcuni ambienti possono richiedere permessi elevati se la directory di installazione non è scrivibile.

## Quando segnalare un bug

Apri un problema su GitHub quando:

- riesci a riprodurre il problema in modo costante
- il problema persiste dopo i passaggi di ripristino di base
- il problema sembra specifico di Bedrock Nexus piuttosto che un problema generale di configurazione di Windows

Quando segnali, includi:

- versione di Windows
- versione di Bedrock Nexus
- i passaggi esatti che hai seguito
- screenshot o log, se disponibili

Apri i problemi qui: [GitHub Issues](https://github.com/BedrockNexusLauncher/BedrockNexusLauncher/issues)
