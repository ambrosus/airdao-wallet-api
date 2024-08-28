import axios from "axios";
import Redis from "ioredis";
import { CronJob } from "cron";
import { singleton } from "tsyringe";
import { tokenPriceUrl } from "../config";

@singleton()
export class ApiPriceWatcher {
  constructor(private readonly cacheStorage: Redis) {
  }

  async run() {
    const job = new CronJob("*/5 * * * *", async () => this.watchApiPrice());
    job.start();
  }

  private async watchApiPrice() {
    console.log("watching api price");
    const { data: { data: { price_usd: price } } } = await axios.get(tokenPriceUrl);
    await this.cacheStorage.set("apiPrice", Number(price));
  }
}