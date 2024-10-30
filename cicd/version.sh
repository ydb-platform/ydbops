APP_VERSION=$(cat ./CHANGELOG.md | grep "##" | head -n 1 | cut -d ' ' -f 2)
echo "v$APP_VERSION"
