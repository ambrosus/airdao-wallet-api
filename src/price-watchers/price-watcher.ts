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
    const watchers = await this.watcherService.getAllWatchers({ priceNotification: "ON" });
    if (!watchers.length) {
      return;
    }

    await Promise.all(watchers.map(async (watcher: Watcher) => {
      const tokenPrice = await this.cacheStorage.get("apiPrice");
      if (!tokenPrice) {
        throw new Error("Price data not found");
      }

      const currentPrice = Number(tokenPrice);

      if (!watcher.tokenPrice) {
        console.log();
        return;
      }

      const percentage: number = (currentPrice - watcher.tokenPrice) / watcher.tokenPrice * 100;

      const roundedPercentage: number = Math.abs((Math.round(percentage * 100) / 100));
      if (roundedPercentage < watcher.threshold) {
        return;
      }

      const roundedPrice: string = currentPrice.toFixed(5);

      const title = notificationsTitleConfig[appEnv].priceAlert;
      const data = { type: "price-alert", percentage: roundedPercentage };
      let body = "";


      if (roundedPercentage >= watcher.threshold) {
        body = `🚀 AMB Price changed on +${roundedPercentage}%! Current price $${roundedPrice}`;
      } else if (roundedPercentage <= -watcher.threshold) {
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
          title: data.type,
          body,
          sent: true,
          timestamp: Date.now()
        }),
      ]);
    }));
  }
}
