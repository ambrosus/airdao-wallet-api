import { CronJob } from "cron";
import { singleton } from "tsyringe";
import axios from "axios";
import Redis from "ioredis";
import { cgTokenPriceUrl } from "../config";

@singleton()
export class CgPriceWatcher {
    private readonly cacheStorage: Redis;

    constructor(cacheStorage: Redis) {
        this.cacheStorage = cacheStorage;
    }
    async run() {
        const job = new CronJob("0 */12 * * *", async () => this.watchCgPrice());
        job.start();
    }

    private async watchCgPrice() {
        const {data: price} = await axios.get(cgTokenPriceUrl);
        await this.cacheStorage.set("cgPrices", JSON.stringify(price.prices));
    }
}
