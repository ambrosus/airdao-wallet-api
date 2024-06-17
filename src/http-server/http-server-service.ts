
// express server
import express from "express";
import cors from "cors";
import {IHandlers} from "./http-server.handlers";


class HttpServerService {
    private app: express.Application;
    private handlers: IHandlers;
    constructor(handlers: IHandlers) {
        this.app = express();
        this.app.use(cors());
        this.app.use(express.json());
        this.app.use(express.urlencoded({ extended: true }));
        this.handlers = handlers;
    }

    init() {
        this.app.get("/watcher/:token", this.handlers.getWatcher.bind(this.handlers));
        this.app.get("/watcher-historical-prices", this.handlers.getWatcherHistoryPrices.bind(this.handlers));

        this.app.post("/watcher", this.handlers.createWatcher.bind(this.handlers));
        this.app.put("/watcher", this.handlers.updateWatcher.bind(this.handlers));

        this.app.delete("/watcher", this.handlers.deleteWatcher.bind(this.handlers));
        this.app.delete("/watcher-addresses", this.handlers.deleteWatcherAddresses.bind(this.handlers));

        this.app.post("/explorer-callback", this.handlers.watcherCallback.bind(this.handlers));

        this.app.put("/push-token", this.handlers.updateWatcherPushToken.bind(this.handlers));
    }

    start(port: number) {
        this.app.listen(port, () => {
        console.log(`Server is running on http://localhost:${port}`);
        });
    }
}
