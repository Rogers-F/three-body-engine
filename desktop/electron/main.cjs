const { app, BrowserWindow } = require('electron')
const { spawn } = require('child_process')
const path = require('path')
const fs = require('fs')
const net = require('net')

const ENGINE_PORT = 9800
let engineProcess = null
let mainWindow = null

function getEnginePath() {
  const ext = process.platform === 'win32' ? '.exe' : ''
  const name = `threebody${ext}`

  // Packaged: extraResources are placed in resources/
  const packed = path.join(process.resourcesPath, name)
  if (fs.existsSync(packed)) return packed

  // Dev: engine/ sibling directory
  const dev = path.join(__dirname, '..', '..', 'engine', name)
  if (fs.existsSync(dev)) return dev

  return null
}

function getConfigPath(engineDir) {
  const candidate = path.join(engineDir, 'config.json')
  if (fs.existsSync(candidate)) return candidate
  return null
}

function startEngine() {
  const enginePath = getEnginePath()
  if (!enginePath) {
    console.error('Engine binary not found')
    return false
  }

  const engineDir = path.dirname(enginePath)
  const configPath = getConfigPath(engineDir)
  const args = configPath ? ['--config', configPath] : []

  engineProcess = spawn(enginePath, args, {
    cwd: engineDir,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  engineProcess.stdout.on('data', (d) => process.stdout.write(`[engine] ${d}`))
  engineProcess.stderr.on('data', (d) => process.stderr.write(`[engine] ${d}`))
  engineProcess.on('exit', (code) => {
    console.log(`Engine exited (code ${code})`)
    engineProcess = null
  })

  return true
}

function waitForEngine(timeout = 15000) {
  const start = Date.now()
  return new Promise((resolve, reject) => {
    function attempt() {
      if (Date.now() - start > timeout) {
        return reject(new Error('Engine start timeout'))
      }
      const socket = net.createConnection({ port: ENGINE_PORT, host: '127.0.0.1' }, () => {
        socket.destroy()
        resolve()
      })
      socket.on('error', () => {
        setTimeout(attempt, 300)
      })
    }
    attempt()
  })
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 900,
    minHeight: 600,
    title: 'Three-Body Engine',
    backgroundColor: '#131314',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
    },
  })

  mainWindow.loadURL(`http://127.0.0.1:${ENGINE_PORT}`)

  mainWindow.on('closed', () => {
    mainWindow = null
  })
}

app.whenReady().then(async () => {
  const started = startEngine()
  if (started) {
    try {
      await waitForEngine()
    } catch (e) {
      console.error(e.message)
    }
  }
  createWindow()
})

app.on('window-all-closed', () => {
  if (engineProcess) {
    engineProcess.kill()
    engineProcess = null
  }
  app.quit()
})

app.on('before-quit', () => {
  if (engineProcess) {
    engineProcess.kill()
    engineProcess = null
  }
})
