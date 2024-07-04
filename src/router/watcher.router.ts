import { container } from "tsyringe";
import { Application } from "express";
import { WatcherNetwork } from "../watcher";

const watcherNetwork = container.resolve(WatcherNetwork);

const routes = (app: Application) => {
    app.get(
        "/api/v1/watcher/:token",
        watcherNetwork.getWatcher.bind(watcherNetwork)
    );

    app.get(
        "/api/v1/watcher-historical-prices",
        watcherNetwork.getWatcherHistoricalPrices.bind(watcherNetwork)
    );

    app.post(
        "/api/v1/watcher",
        watcherNetwork.createWatcher.bind(watcherNetwork)
    );

    app.put(
        "/api/v1/watcher",
        watcherNetwork.updateWatcher.bind(watcherNetwork)
    );

    app.delete(
        "/api/v1/watcher",
        watcherNetwork.deleteWatcher.bind(watcherNetwork)
    );

    app.delete(
        "/api/v1/watcher-addresses",
        watcherNetwork.deleteWatcherAddresses.bind(watcherNetwork)
    );

    app.post(
        "/api/v1/explorer-callback",
        watcherNetwork.watcherCallback.bind(watcherNetwork)
    );

    app.put(
        "/api/v1/push-token",
        watcherNetwork.updateWatcherPushToken.bind(watcherNetwork)
    );
};
export default routes;