#!/bin/bash

# PLC Connection Diagnostic Script

echo "═══════════════════════════════════════════════════════════"
echo "║ PLC Connection Diagnostic"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Read config
if [ -f "config.yaml" ]; then
    TARGET=$(grep "target:" config.yaml | awk '{print $2}' | tr -d '"')
    PLC_IP=$(echo $TARGET | cut -d: -f1)
    PLC_PORT=$(echo $TARGET | cut -d: -f2)
    echo "📝 Config file: config.yaml"
    echo "   Target: $TARGET"
    echo "   IP: $PLC_IP"
    echo "   Port: $PLC_PORT"
else
    echo "❌ No config.yaml found"
    exit 1
fi

echo ""
echo "Testing network connectivity..."
echo ""

# Test ping
echo "1️⃣  Testing ICMP (ping)..."
if ping -c 2 -W 2 $PLC_IP >/dev/null 2>&1; then
    echo "   ✅ PLC responds to ping"
else
    echo "   ⚠️  PLC does not respond to ping (may have ICMP disabled)"
fi

# Test TCP port
echo ""
echo "2️⃣  Testing TCP port $PLC_PORT..."
if nc -z -w 2 $PLC_IP $PLC_PORT 2>/dev/null; then
    echo "   ✅ Port $PLC_PORT is open"
else
    echo "   ❌ Port $PLC_PORT is not reachable"
fi

# Check if server is running
echo ""
echo "3️⃣  Checking if middleware server is running..."
if lsof -i :8080 >/dev/null 2>&1; then
    echo "   ⚠️  Server is already running on port 8080"
    lsof -i :8080 | grep LISTEN
else
    echo "   ✅ Port 8080 is available"
fi

# Try to connect
echo ""
echo "4️⃣  Testing ADS connection..."
cd "$(dirname "$0")"
if [ -f "./middleware-server" ]; then
    echo "   Attempting connection (will timeout after 5 seconds)..."
    timeout 5 ./middleware-server -config config.yaml 2>&1 | grep -E "(Connected|Failed|Error)" | head -5
else
    echo "   ⚠️  middleware-server binary not found - run 'go build' first"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "║ Diagnostic Complete"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📋 Recommendations:"
echo ""
if ! ping -c 1 -W 2 $PLC_IP >/dev/null 2>&1 && ! nc -z -w 2 $PLC_IP $PLC_PORT 2>/dev/null; then
    echo "❌ PLC is not reachable at $PLC_IP:$PLC_PORT"
    echo ""
    echo "   Possible causes:"
    echo "   • PLC is powered off"
    echo "   • PLC IP address has changed"
    echo "   • Network connection issue"
    echo "   • Firewall blocking connection"
    echo ""
    echo "   Next steps:"
    echo "   1. Verify PLC is powered on and running"
    echo "   2. Check PLC's current IP address in TwinCAT"
    echo "   3. Update config.yaml with correct IP if needed"
    echo "   4. Check network connectivity"
elif ! nc -z -w 2 $PLC_IP $PLC_PORT 2>/dev/null; then
    echo "⚠️  PLC responds but port $PLC_PORT is not accessible"
    echo ""
    echo "   Possible causes:"
    echo "   • TwinCAT is not running"
    echo "   • ADS router is not started"
    echo "   • Wrong port number in config"
    echo ""
    echo "   Next steps:"
    echo "   1. Start TwinCAT on the PLC"
    echo "   2. Verify ADS port (usually 851 or 48898)"
else
    echo "✅ PLC appears to be reachable"
    echo ""
    echo "   If connection still fails:"
    echo "   1. Check AMS Net ID configuration"
    echo "   2. Verify routes are configured in TwinCAT"
    echo "   3. Check firewall settings"
fi
echo ""
