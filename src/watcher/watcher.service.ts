import {WatcherRepository} from "./watcher.repository";
import Redis from "ioredis";
import {Watcher} from "./watcher.model";
import {ExplorerService} from "../explorer";

export class WatcherService {

    constructor(
        private readonly watcherRepository: WatcherRepository,
        private readonly cacheStorage: Redis,
        private readonly explorerService: ExplorerService,
    ) {}

    async getWatcher(pushToken: string) {
        return await this.watcherRepository.getWatcher(pushToken);
    }

    async getAllWatchers(filter?: Record<string, string>) {
        return await this.watcherRepository.getAllWatchers(filter);
    }
    async getWatcherHistoryPrices() {
        return this.cacheStorage.get("cgPrices");
    }

    async createWatcher(pushToken: string, deviceId?: string) {
        if(deviceId) {
            const watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
            if(watcher) {
                await this.watcherRepository.deleteWatcher({deviceId});
            }
        }

        const encodedPushToken = Buffer.from(pushToken).toString("base64");
        const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
        if (watcher) {
            throw new Error("watcher for this address and token already exists");
        }

        const tokenPrice = await this.cacheStorage.get("apiPrice");
        if (!tokenPrice) {
            throw new Error("Price data not found");
        }


        await this.watcherRepository.createWatcher({
            pushToken: encodedPushToken,
            threshold: 5,
            txNotification: "ON",
            priceNotification: "ON",
            tokenPrice: Number(tokenPrice),
            deviceId,
        } as unknown as Watcher);
    }
    async updateWatcher(pushToken: string, addresses: string[], threshold: number, txNotification: string, priceNotification: string) {
        const watcher = await this.watcherRepository.getWatcher(pushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }

        if (addresses.length > 0) {
            const uniqueAddresses = addresses.filter(address => !watcher.addresses?.includes(address)) || [];

            if (uniqueAddresses.length > 0) {
                await this.explorerService.subscribeAddresses(uniqueAddresses);
                await this.watcherRepository.updateWatcher({pushToken}, {addresses: uniqueAddresses});
            }
        }
        if (threshold) {
            await this.watcherRepository.updateWatcher({pushToken}, {threshold});
        }
        if (txNotification) {
            await this.watcherRepository.updateWatcher({pushToken}, {txNotification});
        }
        if (priceNotification) {
            await this.watcherRepository.updateWatcher({pushToken}, {priceNotification});
        }
    }

    async updateWatcherPrice(pushToken: string, price: number) {
        await this.watcherRepository.updateWatcher({pushToken}, {tokenPrice: price});

    }
    async deleteWatcher() {

    }
    async deleteWatcherAddresses() {

    }
    async watcherCallback() {

    }
    async updateWatcherPushToken() {

    }
}
