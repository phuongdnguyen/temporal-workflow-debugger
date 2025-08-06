import http from "node:http"
import { AddressInfo } from "node:net"
import express from "express"
import { HistoryDebuggerPanel } from "./panel"

async function listen(server: http.Server, port: number, hostname?: string): Promise<http.Server> {
  await new Promise<void>((resolve, reject) => {
    if (hostname) {
      server.listen(port, hostname, resolve)
    } else {
      server.listen(port, resolve)
    }
    server.once("error", reject)
  })
  return server
}

function mustBeAddrInfo(info: string | AddressInfo | null): asserts info is AddressInfo {
  if (info === null) {
    throw new TypeError("Expected AddressInfo got null")
  }
  if (typeof info === "string") {
    throw new TypeError("Expected AddressInfo got a string")
  }
}

export class Server {
  static async create(address = "127.0.0.1", port = 54578): Promise<Server> {
    const app = express()
    app.use(express.json())
    app.get("/history", async (_req, res) => {
      try {
        const { currentHistoryBuffer } = await HistoryDebuggerPanel.instance
        if (!currentHistoryBuffer) {
          res.status(404).json({ error: "No current history available" })
          return
        }
        // Set Content-Type to match Java extension (application/octet-stream for raw bytes)
        res.setHeader("Content-Type", "application/octet-stream")
        res.end(currentHistoryBuffer)
      } catch (error) {
        res.status(500).json({ error: `${error}` })
      }
    })
    app.post("/current-event", async (req, res) => {
      if (!(typeof req.body === "object" && typeof req.body.eventId === "number")) {
        res.status(400).send({ error: "Bad request" })
        return
      }
      const { eventId } = req.body
      try {
        const instance = await HistoryDebuggerPanel.instance
        await instance.updateCurrentWFTStarted(eventId)
      } catch (error) {
        res.status(500).send({ error: `${error}` })
        return
      }
      res.end()
    })
    app.get("/breakpoints", async (_req, res) => {
      try {
        const instance = await HistoryDebuggerPanel.instance
        const breakpoints = instance.getEnabledBreakpoints()
        res.status(200).json({
          breakpoints: breakpoints
        })
      } catch (error) {
        res.status(500).json({ error: `${error}` })
      }
    })
    const server = new http.Server(app)
    await listen(server, port, address)
    return new this(server)
  }

  constructor(protected readonly server: http.Server) { }

  get url(): string {
    const addr = this.server.address()
    mustBeAddrInfo(addr)
    return `http://${addr.address}:${addr.port}`
  }

  terminate(): void {
    console.log(`Closing server on ${this.url}`)
    this.server.close()
  }
}
