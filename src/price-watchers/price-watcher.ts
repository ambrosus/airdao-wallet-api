import { CronJob } from "cron";
import { singleton } from "tsyringe";
import Redis from "ioredis";
import { NotificationService } from "../notification-sender";
import {WatcherService} from "../watcher/watcher.service";

@singleton()
export class PriceWatcher {
    private readonly cacheStorage: Redis;
    private readonly watcherService: WatcherService;
    private readonly notificationService: NotificationService;

    constructor(cacheStorage: Redis, watcherService: WatcherService, notificationService: NotificationService) {
        this.cacheStorage = cacheStorage;
        this.watcherService = watcherService;
        this.notificationService = notificationService;
    }
    async run() {
        const job = new CronJob("*/330 * * * * *", async () => this.watchPrice());
        job.start();
    }

    private async watchPrice() {
        const watchers = await this.watcherService.getAllWatchers();
        await Promise.all(watchers.map(async (watcher) => {
            const tokenPrice = await this.cacheStorage.get("apiPrice");
            if (!tokenPrice) {
                throw new Error("Price data not found");
            }

            const percentage: number = (Number(tokenPrice) - watcher.tokenPrice!) / watcher.tokenPrice! * 100;
            const roundedPercentage: number = Math.abs((Math.round(percentage * 100) / 100));
            const roundedPrice: string = Number(tokenPrice).toFixed(5);

            if (watcher.txNotification === "ON") {
                const title = "Price Alert";
                const data = {type: "price-alert", percentage: roundedPercentage};
                let body;
                

                if (roundedPercentage >= watcher.threshold) {
                    body = `🚀 AMB Price changed on +${roundedPercentage}%! Current price $${roundedPrice}`;
                } else if (roundedPercentage <= -watcher.threshold) {
                    body = `🔻 AMB Price changed on -${roundedPercentage}%! Current price $${roundedPrice}`;
                }

                await this.notificationService.sendNotification({title, body: body as unknown as string, pushToken: watcher.pushToken, data});
            }
            await this.watcherService.updateWatcherPrice(watcher.pushToken, Number(tokenPrice));
        }));
    }
}
