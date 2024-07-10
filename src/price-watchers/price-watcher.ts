import Redis from "ioredis";
import { CronJob } from "cron";
import { singleton } from "tsyringe";
import { WatcherService } from "../watcher";
import { NotificationService } from "../notification-sender";

@singleton()
export class PriceWatcher {
    constructor(
        private readonly cacheStorage: Redis,
        private readonly watcherService: WatcherService,
        private readonly notificationService: NotificationService
    ) {
    }
    async run() {
        const job = new CronJob("*/330 * * * * *", async () => this.watchPrice());
        job.start();
    }

    private async watchPrice() {
        console.log("Watching Price");
        const watchers = await this.watcherService.getAllWatchers({ priceNotification: "ON" });
        console.log("WATCHERS", watchers);
        if (!watchers.length) return;
        await Promise.all(watchers.map(async (watcher) => {
            const tokenPrice = await this.cacheStorage.get("apiPrice");
            console.log("TOKEN PRICE", tokenPrice);
            if (!tokenPrice) {
                throw new Error("Price data not found");
            }

            const currentPrice = Number(tokenPrice);
            console.log("Current Price", currentPrice);

            const percentage: number = (currentPrice - watcher.tokenPrice!) / watcher.tokenPrice! * 100;
            console.log("Percentage", percentage);

            const roundedPercentage: number = Math.abs((Math.round(percentage * 100) / 100));
            console.log("Rounded Percentage", percentage);

            const roundedPrice: string = currentPrice.toFixed(5);
            console.log("roundedPrice", roundedPrice);


            const title = "Price Alert";
            const data = { type: "price-alert", percentage: roundedPercentage };
            let body = "";


            if (roundedPercentage >= watcher.threshold) {
                body = `🚀 AMB Price changed on +${roundedPercentage}%! Current price $${roundedPrice}`;
            } else if (roundedPercentage <= -watcher.threshold) {
                body = `🔻 AMB Price changed on -${roundedPercentage}%! Current price $${roundedPrice}`;
            }

            const decodedPushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");
            console.log("decodedPushToken", decodedPushToken);


            console.log("SENDING PRICE NOTIFICATION", { title, body, decodedPushToken, data });
            await Promise.all([
                this.notificationService.sendNotification({
                    title,
                    body,
                    pushToken: decodedPushToken,
                    data
                }),
                this.watcherService.updateWatcherPrice(watcher.pushToken, currentPrice)
            ]);
        }));
    }
}
