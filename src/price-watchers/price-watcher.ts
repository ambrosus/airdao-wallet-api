import Redis from "ioredis";
import { singleton } from "tsyringe";
import { Watcher, WatcherService } from "../watcher";
import { NotificationService } from "../notification-sender";
import { HistoricalNotificationsService } from "../historical-notifications";
import { appEnv, notificationsTitleConfig } from "../config";

@singleton()
export class PriceWatcher {
  constructor(
    private readonly cacheStorage: Redis,
    private readonly watcherService: WatcherService,
    private readonly notificationService: NotificationService,
    private readonly historicalNotificationsService: HistoricalNotificationsService,
  ) {
  }

  async run() {
    this.scheduleNextRun();
  }

  private scheduleNextRun() {
    setTimeout(async () => {
      await this.watchPrice();
      this.scheduleNextRun();
    }, 330 * 1000);
  }

  private async watchPrice() {
    console.log("Watching Price");
    const watchers = await this.watcherService.getAllWatchers({ priceNotification: { $regex: /^on$/i } });
    if (!watchers.length) {
      return;
    }

    await Promise.all(watchers.map(async (watcher: Watcher) => {
      const tokenPrice = await this.cacheStorage.get("apiPrice");
      if (!tokenPrice) {
        throw new Error("Price data not found");
      }

      const currentPrice = Number(tokenPrice);
      console.log("Current Price", currentPrice);
      console.log("Watcher Token Price", watcher.tokenPrice);

      if (!watcher.tokenPrice) return;

      const percentage: number = (currentPrice - watcher.tokenPrice) / watcher.tokenPrice * 100;
      console.log("Price change", percentage);

      const roundedPercentage: number = Math.abs(Math.round(percentage * 100) / 100);
      console.log("Rounded Price change", roundedPercentage);
      console.log("Watcher Threshold", watcher.threshold);
      if (roundedPercentage < watcher.threshold) return;

      const roundedPrice: string = currentPrice.toFixed(5);
      console.log("Current Price", roundedPrice);

      const title = notificationsTitleConfig[appEnv].priceAlert;
      const data = { type: "price-alert", percentage: roundedPercentage };
      let body = "";

      if (percentage >= watcher.threshold) {
        body = `🚀 AMB Price changed on +${roundedPercentage}%! Current price $${roundedPrice}`;
      } else if (percentage <= -watcher.threshold) {
        body = `🔻 AMB Price changed on -${roundedPercentage}%! Current price $${roundedPrice}`;
      }

      const decodedPushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");

      return Promise.all([
        this.notificationService.sendNotification({
          title,
          body,
          pushToken: decodedPushToken,
          data
        }),
        this.watcherService.updateWatcherPrice(watcher.pushToken, currentPrice),
        this.historicalNotificationsService.addHistoricalNotification(watcher._id, {
          title,
          body,
          sent: true,
          timestamp: Date.now()
        }),
      ]);
    }));
  }
}
