import { CronJob } from "cron";
import { singleton } from "tsyringe";
import axios from "axios";
import Redis from "ioredis";
import {tokenPriceUrl} from "../config";

@singleton()
export class ApiPriceWatcher {
    private readonly cacheStorage: Redis;

    constructor(cacheStorage: Redis) {
        this.cacheStorage = cacheStorage;
    }

    async run() {
        const job = new CronJob("*/5 * * * *", async () => this.watchApiPrice());
        job.start();
    }

    private async watchApiPrice() {
        const {data: price} = await axios.get(tokenPriceUrl);
        await this.cacheStorage.set("apiPrice", price.data.PriceUSD);
    }
}