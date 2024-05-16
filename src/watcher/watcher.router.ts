import {Application} from "express";
import {container} from "tsyringe";
import {WatcherNetwork} from "./watcher.network";
// todo move to httpServer service

const watcherNetwork = container.resolve(WatcherNetwork)

const routes = (app: Application) => {
    app.get("/watcher/:token", watcherNetwork.getWatcher.bind(watcherNetwork))
    app.get("/watcher-historical-prices", watcherNetwork.getWatcherHistoryPrices.bind(watcherNetwork))

    app.post("/watcher", watcherNetwork.createWatcher.bind(watcherNetwork))
    app.put("/watcher", watcherNetwork.updateWatcher.bind(watcherNetwork))

    app.delete("/watcher", watcherNetwork.deleteWatcher.bind(watcherNetwork))
    app.delete("/watcher-addresses", watcherNetwork.deleteWatcherAddresses.bind(watcherNetwork))

    app.post("/explorer-callback", watcherNetwork.watcherCallback.bind(watcherNetwork))

    app.put("/push-token", watcherNetwork.updateWatcherPushToken.bind(watcherNetwork))
}
