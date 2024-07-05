import axios from "axios";
import Redis from "ioredis";
import { CronJob } from "cron";
import { singleton } from "tsyringe";
import { cgTokenPriceUrl } from "../config";

@singleton()
export class CgPriceWatcher {
    constructor(private readonly cacheStorage: Redis) {
    }

    async run() {
        const job = new CronJob("0 */12 * * *", async () => this.watchCgPrice());
        job.start();
    }

    private async watchCgPrice() {
        const { data: { prices } } = await axios.get(cgTokenPriceUrl);
        console.log("Setting Cg Prices", JSON.stringify(prices));
        await this.cacheStorage.set("cgPrices", JSON.stringify(prices));
    }
}
