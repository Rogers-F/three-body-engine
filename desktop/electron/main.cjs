const { app, BrowserWindow } = require('electron')
const { spawn } = require('child_process')
const path = require('path')
const fs = require('fs')
const net = require('net')

const ENGINE_PORT = 9800
let engineProcess = null
let mainWindow = null

// ── Locate the Go engine binary ────────────────────────────
function findEngine() {
  const ext = process.platform === 'win32' ? '.exe' : ''
  const name = `threebody${ext}`
  const candidates = [
    // Packaged (electron-builder extraResources)
    path.join(process.resourcesPath || '', name),
    // Dev: engine/ sibling
    path.join(__dirname, '..', '..', 'engine', name),
  ]
  for (const p of candidates) {
    if (fs.existsSync(p)) return p
  }
  return null
}

// ── Locate the frontend dist/ directory ────────────────────
function findDist() {
  const candidates = [
    // Packaged (electron-builder files)
    path.join(__dirname, '..', 'dist', 'index.html'),
    // Dev: desktop/dist after `npm run build`
    path.join(__dirname, '..', 'dist', 'index.html'),
  ]
  for (const p of candidates) {
    if (fs.existsSync(p)) return path.dirname(p)
  }
  return null
}

// ── Start the Go engine subprocess ─────────────────────────
function startEngine() {
  const enginePath = findEngine()
  if (!enginePath) {
    console.error('[electron] Engine binary not found, running frontend-only')
    return
  }

  const engineDir = path.dirname(enginePath)
  const configPath = path.join(engineDir, 'config.json')
  const args = fs.existsSync(configPath) ? ['--config', configPath] : []

  console.log(`[electron] Starting engine: ${enginePath}`)
  engineProcess = spawn(enginePath, args, {
    cwd: engineDir,
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  engineProcess.stdout.on('data', (d) => process.stdout.write(String(d)))
  engineProcess.stderr.on('data', (d) => process.stderr.write(String(d)))
  engineProcess.on('exit', (code) => {
    console.log(`[electron] Engine exited (${code})`)
    engineProcess = null
  })
}

// ── Wait for Go engine to accept connections ───────────────
function waitForEngine(timeoutMs = 10000) {
  return new Promise((resolve, reject) => {
    const deadline = Date.now() + timeoutMs
    ;(function probe() {
      if (Date.now() > deadline) return reject(new Error('engine timeout'))
      const sock = net.createConnection({ port: ENGINE_PORT, host: '127.0.0.1' }, () => {
        sock.destroy()
        resolve()
      })
      sock.on('error', () => setTimeout(probe, 200))
    })()
  })
}

// ── Create the main window ─────────────────────────────────
function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 900,
    minHeight: 600,
    title: 'Three-Body Engine',
    backgroundColor: '#131314',
    show: false,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  })

  const distDir = findDist()
  if (distDir) {
    console.log(`[electron] Loading frontend from ${distDir}`)
    mainWindow.loadFile(path.join(distDir, 'index.html'))
  } else {
    // Fallback: load from Go engine server
    console.log('[electron] dist/ not found, loading from engine server')
    mainWindow.loadURL(`http://127.0.0.1:${ENGINE_PORT}`)
  }

  mainWindow.once('ready-to-show', () => mainWindow.show())
  mainWindow.on('closed', () => { mainWindow = null })
}

// ── App lifecycle ──────────────────────────────────────────
app.whenReady().then(async () => {
  startEngine()
  if (engineProcess) {
    try { await waitForEngine() } catch (e) {
      console.error(`[electron] ${e.message}`)
    }
  }
  createWindow()
})

app.on('window-all-closed', () => {
  killEngine()
  app.quit()
})

app.on('before-quit', killEngine)

function killEngine() {
  if (engineProcess) {
    engineProcess.kill()
    engineProcess = null
  }
}
