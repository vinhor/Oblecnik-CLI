# Oblečník CLI

Oblečník je aplikace pro získávání počasí a doporučování oblečení podle toho. Toto je CLI.

## Instalace

Jelikož lokace je v kódu (prozatím), musíte naklonovat repozitář, změnit kód, a zkompilovat.

1. Nejprve nainstalujte [Go](https://go.dev/)
2. Naklonujte repozitář s `git clone https://github.com/vinhor/Oblecnik-CLI`, nebo ho stáhněte jako zip.
3. V souboru `oblecnik.go` upravte konstanty `lat`, `lon` a `alt` na vaše lokaci. Pokud neznáte vaši nadmořskou býšku (alt), nastavte ji na -500.
4. Zkompilujte aplikaci pomocí `go build`.
5. Výsledný soubour (oblecnik nebo oblecnik.exe) buď spustěte, nebo si ho dejte do vámi preferované složky.
