#!/bin/bash
# Builds the Moly screens (the extension's sidebar) for old Android web views (ES2015) and puts them, with the phone shim, in www/.
set -euo pipefail
cd "$(dirname "$0")/.."
EXT=../moly-extension
[ -d "$EXT/node_modules" ] || (cd "$EXT" && npm ci)
rm -rf build/ext www
(cd "$EXT" && npx vite build --target es2015 --outDir "$OLDPWD/build/ext" --emptyOutDir >/dev/null)
mkdir -p www
cp -r build/ext/sidebar.js build/ext/chunk-*.js build/ext/style.css build/ext/images www/
# the shim reports the product version to the screens (About, update checks): it is written here from installer/VERSION, never typed twice
V=$(tr -d ' \n' < ../installer/VERSION)
[[ "$V" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "installer/VERSION must be MAJOR.MINOR.PATCH, found '$V'" >&2; exit 1; }
sed "s/__MOLY_VERSION__/$V/" shim.js > www/shim.js
grep -q "__MOLY_VERSION__" www/shim.js && { echo "the version was not written into www/shim.js" >&2; exit 1; }
cat > www/index.html <<'HTML'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover">
  <title>Moly</title>
  <link rel="stylesheet" href="./style.css">
  <script src="./shim.js"></script>
</head>
<body>
  <div id="root"></div>
  <script type="module" src="./sidebar.js"></script>
</body>
</html>
HTML
echo "screens built in www/"
