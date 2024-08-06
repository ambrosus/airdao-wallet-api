import { DependencyContainer } from "tsyringe";
import { Application } from "express";
import { WatcherNetwork } from "../watcher";
import { NotificationService } from "../notification-sender";


const routes = (app: Application, container: DependencyContainer) => {

    const watcherNetwork = container.resolve(WatcherNetwork);

    app.get(
        "/api/v1/watcher/:pushToken",
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

    app.post("/api/v1/send-notification", (req, res) => {
        const { title, body, pushToken, data } = req.body;
        container.resolve(NotificationService).sendNotification({
            title,
            body,
            pushToken,
            data
        }).then((result) => {
            res.send(result);
        }).catch((error) => {
            res.status(500).send(error);
        });
    });
};
export default routes;