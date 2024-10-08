import { DependencyContainer } from "tsyringe";
import { Application } from "express";
import { WatcherNetwork } from "../watcher";
import { NotificationService } from "../notification-sender";
import { HistoricalNotificationsService } from "../historical-notifications";

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

  app.post("/api/v1/send-notification", async (req, res) => {
    const { title, body, pushToken, data, watcherId } = req.body;
    try {
      const result = await container.resolve(NotificationService).sendNotification({
        title,
        body,
        pushToken,
        data
      });
      await container.resolve(HistoricalNotificationsService).addHistoricalNotification(watcherId, {
        title,
        body,
        sent: true,
        timestamp: Date.now()
      });
      res.send(result);
    } catch (e) {
      res.status(500).send(e);
    }
  });
};

export default routes;