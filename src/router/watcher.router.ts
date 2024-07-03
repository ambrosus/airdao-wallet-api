import { container } from "tsyringe";
import { Application } from "express";
import { WatcherNetwork } from "../watcher";

const watcherNetwork = container.resolve(WatcherNetwork);

const routes = (app: Application) => {
    app.get(
        "/watcher/:token",
        watcherNetwork.getWatcher.bind(watcherNetwork)
    );

    app.get(
        "/watcher-historical-prices",
        watcherNetwork.getWatcherHistoricalPrices.bind(watcherNetwork)
    );

    app.post(
        "/watcher",
        watcherNetwork.createWatcher.bind(watcherNetwork)
    );

    app.put(
        "/watcher",
        watcherNetwork.updateWatcher.bind(watcherNetwork)
    );

    app.delete(
        "/watcher",
        watcherNetwork.deleteWatcher.bind(watcherNetwork)
    );

    app.delete(
        "/watcher-addresses",
        watcherNetwork.deleteWatcherAddresses.bind(watcherNetwork)
    );

    app.post(
        "/explorer-callback",
        watcherNetwork.watcherCallback.bind(watcherNetwork)
    );

    app.put(
        "/push-token",
        watcherNetwork.updateWatcherPushToken.bind(watcherNetwork)
    );
};


//	router.Get("/watcher/:token", h.GetWatcherHandler)
// 	router.Get("/watcher-historical-prices", h.GetWatcherHistoryPricesHandler)
//
// 	router.Post("/watcher", h.CreateWatcherHandler)
// 	router.Put("/watcher", h.UpdateWatcherHandler)
//
// 	router.Delete("/watcher", h.DeleteWatcherHandler)
// 	router.Delete("/watcher-addresses", h.DeleteWatcherAddressesHandler)
//
// 	router.Post("/explorer-callback", h.WatcherCallbackHandler)
//
// 	router.Put("/push-token", h.UpdateWatcherPushTokenHandler)
export default routes;