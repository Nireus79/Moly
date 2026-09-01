# Moly Troubleshooting Guide

Common issues and solutions for Moly installation and usage.

## Installation Issues

### "Node.js not found"
**Error:** `Node.js 16+ required`

**Solution:**
1. Download from https://nodejs.org/
2. Install the LTS version (18.x or later)
3. Verify installation: `node -v`
4. Re-run installer: `moly-installer`

### "System requirements not met"
**Error:** Installer shows critical issues (RAM, disk, OS)

**Solution:**
- **RAM:** You need at least 4GB RAM (8GB recommended)
  - Check available RAM: `free -h` (Linux), Activity Monitor (macOS), Task Manager (Windows)
  - Close unnecessary applications

- **Disk Space:** You need 10GB free
  - Check disk space: `df -h` (Linux/macOS), `dir` (Windows)
  - Clean up disk or use cloud-only mode

- **OS:** Must be Windows 10+, macOS 10.15+, or Linux
  - Check your OS version and upgrade if needed

- **CPU:** Minimum 2 cores recommended
  - System may work with 1 core but will be slow

### "Download failed"
**Error:** Installer cannot download Ollama or LM Studio

**Solution:**
1. Check internet connection: `ping google.com`
2. Try installer again - may be temporary network issue
3. Download manually:
   - Ollama: https://ollama.ai/download
   - LM Studio: https://lmstudio.ai/
4. Install manually, then run `moly-installer` to configure

### "npm install fails"
**Error:** `npm ERR! ERESOLVE unable to resolve dependency tree`

**Solution:**
```bash
# Clear npm cache
npm cache clean --force

# Install installer again
npm install -g moly-installer

# Or use npm version 8+
npm install -n 8 -g npm
```

### "Permission denied on Linux"
**Error:** `Error: EACCES: permission denied`

**Solution:**
```bash
# Fix npm permissions
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
export PATH=~/.npm-global/bin:$PATH

# Add to ~/.bashrc or ~/.zshrc to make permanent
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# Now install installer
npm install -g moly-installer
```

## Ollama Issues

### "Ollama not found" or "ollama command not recognized"
**Error:** Installer can't find Ollama CLI

**Solution:**
1. Verify Ollama is installed: `ollama --version`
2. If not found, install from https://ollama.ai/download
3. Restart terminal/PowerShell after installation
4. Add to PATH if needed:
   - **Windows:** Usually automatic
   - **macOS:** Should be in `/Applications/Ollama.app`
   - **Linux:** Should be in `/usr/bin/ollama`

### "Ollama not responding"
**Error:** `connect ECONNREFUSED 127.0.0.1:11434`

**Solution:**
1. Start Ollama: `ollama serve`
2. Verify it's running:
   ```bash
   curl http://localhost:11434/api/tags
   ```
3. If still not working:
   - Restart Ollama
   - Check if port 11434 is in use: `lsof -i :11434` (macOS/Linux)
   - Use different port: Configure in Ollama settings

### "Model download stuck"
**Error:** `ollama pull mistral` hangs or is very slow

**Solution:**
1. Check internet connection
2. Verify disk space (need ~4GB for Mistral 7B)
3. Try with timeout: `timeout 300 ollama pull mistral`
4. Kill and retry:
   ```bash
   # Stop Ollama
   ps aux | grep ollama
   kill <pid>

   # Restart and try again
   ollama serve
   ollama pull mistral
   ```

### "Port 11434 already in use"
**Error:** `listen EADDRINUSE :::11434`

**Solution:**
```bash
# Find what's using the port
lsof -i :11434          # macOS/Linux
netstat -ano | findstr :11434  # Windows

# Kill the process (if not Ollama)
kill <pid>              # macOS/Linux
taskkill /pid <pid> /f  # Windows

# Or use different port in Ollama config
export OLLAMA_PORT=11440
ollama serve
```

## CORS Proxy Issues

### "CORS Proxy not running"
**Error:** Moly shows CORS error, proxy not responding

**Solution:**
1. Start proxy manually:
   ```bash
   moly-proxy
   ```
2. Verify it's running:
   ```bash
   curl http://localhost:11435/api/tags
   ```
3. Check if port 11435 is in use:
   ```bash
   lsof -i :11435        # macOS/Linux
   netstat -ano | findstr :11435  # Windows
   ```

### "CORS headers not added"
**Error:** Browser shows CORS error even with proxy

**Solution:**
1. Verify Ollama is running: `curl http://localhost:11434/api/tags`
2. Verify proxy is running: `curl http://localhost:11435/api/tags`
3. Check proxy response headers:
   ```bash
   curl -i http://localhost:11435/api/tags
   ```
   Should show: `Access-Control-Allow-Origin: *`

4. Restart both:
   ```bash
   # Terminal 1: Ollama
   ollama serve

   # Terminal 2: Proxy
   moly-proxy
   ```

### "Proxy fails to start"
**Error:** `listen EADDRINUSE :::11435` or other error

**Solution:**
1. Check port availability: `lsof -i :11435`
2. Use different port: `moly-proxy --port 11440`
3. Update Moly settings to use new port
4. Restart proxy

## LM Studio Issues

### "LM Studio not found"
**Error:** LM Studio app not installed

**Solution:**
1. Download from https://lmstudio.ai/
2. Install the application for your OS
3. Launch LM Studio
4. Download model through the app's Search tab
5. Configure Moly to use LM Studio (localhost:8000)

### "Model download stuck in LM Studio"
**Error:** Model download not progressing

**Solution:**
1. Check internet connection
2. Check available disk space (need ~4GB minimum)
3. Try different model
4. Restart LM Studio
5. Check LM Studio logs for errors

## Moly Extension Issues

### "Auto-detection not working"
**Error:** Moly doesn't find local Ollama/LM Studio

**Solution:**
1. Make sure local service is running
2. Check ports are correct:
   - Ollama proxy: 11435
   - LM Studio: 8000
3. Reload Moly extension: `chrome://extensions` → reload
4. Check browser console (F12) for errors

### "Slow suggestions or timeouts"
**Error:** Moly takes 30+ seconds or times out

**Solution:**
1. Check if model is still loading: `ollama list`
2. Reduce context window (shorter previous messages)
3. Try a smaller/faster model
4. Check system resources (RAM, CPU usage)
5. Restart Ollama/model

### "Settings not saving"
**Error:** Provider configuration reverts after reload

**Solution:**
1. Check browser storage permissions
2. Disable privacy mode/incognito
3. Clear browser cache and reload
4. Try different browser
5. Check browser console for JavaScript errors

## Cloud Provider Issues

### "Claude API key invalid"
**Error:** `Error: Invalid API Key` or `401 Unauthorized`

**Solution:**
1. Get key from https://console.anthropic.com/keys
2. Ensure key starts with `sk-ant-`
3. Verify key is not expired or revoked
4. Copy full key (no spaces)
5. Test with curl:
   ```bash
   curl https://api.anthropic.com/v1/models \
     -H "x-api-key: sk-ant-YOUR_KEY"
   ```

### "OpenAI API key invalid"
**Error:** `Error: Incorrect API key provided`

**Solution:**
1. Get key from https://platform.openai.com/api-keys
2. Ensure key starts with `sk-`
3. Verify key is active (not deleted)
4. Ensure billing is set up
5. Test with curl:
   ```bash
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer sk-YOUR_KEY"
   ```

### "Rate limited"
**Error:** `Error: 429 Rate Limit Exceeded`

**Solution:**
1. Wait a few minutes before retrying
2. Reduce number of suggestions (settings)
3. Upgrade API plan if heavy usage
4. Implement request throttling

## Platform-Specific Issues

### Windows

#### "PowerShell Execution Policy"
**Error:** `cannot be loaded because running scripts is disabled`

**Solution:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### "Task Scheduler not creating service"
**Error:** Task Scheduler fails to create Moly Proxy task

**Solution:**
1. Run installer as Administrator
2. Check Task Scheduler permissions
3. Manually create task:
   - Open Task Scheduler
   - Create Basic Task
   - Name: "Moly Proxy"
   - Trigger: At logon
   - Action: Start a program (`moly-proxy`)

### macOS

#### "M1/M2 Chip compatibility"
**Error:** Binary is not compatible or crashes

**Solution:**
1. Ollama has native M1/M2 support
2. Download ARM64 version
3. LM Studio also supports M1/M2
4. If issues, check Activity Monitor for crashes

#### "LaunchAgent not loading"
**Error:** Service doesn't auto-start on login

**Solution:**
```bash
# Load manually
launchctl load ~/Library/LaunchAgents/com.moly.proxy.plist

# Verify it's loaded
launchctl list | grep com.moly.proxy

# Check logs
log stream --predicate 'process contains "moly"'
```

### Linux

#### "systemd service not starting"
**Error:** `systemctl status moly-proxy` shows failed

**Solution:**
```bash
# Check service status
sudo systemctl status moly-proxy

# Check logs
sudo journalctl -u moly-proxy -n 20

# Restart service
sudo systemctl restart moly-proxy

# Enable auto-start
sudo systemctl enable moly-proxy
```

#### "Permission denied for /etc/systemd/system"
**Error:** Cannot write service file

**Solution:**
```bash
# Create service as root
sudo bash -c 'cat > /etc/systemd/system/moly-proxy.service <<EOF
[Unit]
Description=Moly CORS Proxy
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=$(which moly-proxy)
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF'

sudo systemctl daemon-reload
sudo systemctl enable moly-proxy
sudo systemctl start moly-proxy
```

## Diagnostic Steps

### Check Everything is Running

```bash
# 1. Check Node.js
node -v
npm -v

# 2. Check Ollama
curl http://localhost:11434/api/tags

# 3. Check CORS Proxy
curl http://localhost:11435/api/tags

# 4. Check LM Studio
curl http://localhost:8000/api/models

# 5. Check APIs (if using cloud)
curl https://api.anthropic.com/v1/models \
  -H "x-api-key: sk-ant-YOUR_KEY"
```

### Collect Debug Info

```bash
# Ollama
ollama --version
ollama list

# CORS Proxy
moly-proxy --version 2>/dev/null || npm list -g moly-proxy

# System
uname -a                    # macOS/Linux
systeminfo | find "OS"      # Windows
```

### Enable Verbose Logging

```bash
# Ollama debug
OLLAMA_DEBUG=1 ollama serve

# Proxy debug
DEBUG=* moly-proxy

# Check browser console (F12) for Moly extension logs
```

## Still Stuck?

1. Check GitHub Issues: https://github.com/Nireus79/Moly/issues
2. Create new issue with:
   - OS and version
   - Node.js version
   - Installer output (copy full console)
   - Error messages
   - What you tried
3. Enable debug logging and attach logs

## Common Error Messages

| Error | Likely Cause | Solution |
|-------|-------------|----------|
| `ECONNREFUSED` | Service not running | Start Ollama/proxy |
| `EADDRINUSE` | Port in use | Check what's using port |
| `EACCES` | Permission denied | Check file permissions |
| `ETIMEDOUT` | Service too slow/blocked | Restart, check firewall |
| `401/403` | Invalid credentials | Check API key |
| `429` | Rate limited | Wait, then retry |
| `CORS error` | Proxy not working | Verify proxy at port 11435 |
| `Invalid API Key` | Wrong/expired key | Get fresh key from provider |

---

**Last Updated:** 2026-09-01  
**Version:** Moly 1.0  
**Support:** https://github.com/Nireus79/Moly/issues
