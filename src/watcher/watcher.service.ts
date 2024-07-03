import Redis from "ioredis";
import {singleton} from "tsyringe";
import { Watcher } from "./watcher.model";
import { ExplorerService } from "../explorer";
import { WatcherRepository } from "./watcher.repository";

@singleton()
export class WatcherService {

    constructor(
        private readonly watcherRepository: WatcherRepository,
        private readonly cacheStorage: Redis,
        private readonly explorerService: ExplorerService,
    ) {}

    async getWatcher(pushToken: string) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");

        const watcher =  await this.watcherRepository.getWatcher(encodedPushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }

        // TODO: add historical notifications

        return {...watcher, historicalNotifications: []};
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

    async updateWatcher(pushToken: string, updateFields: {addresses?: string[], threshold?: number, txNotification?: string, priceNotification?: string}) {
        const watcher = await this.watcherRepository.getWatcher(pushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }

        const {addresses, threshold, txNotification, priceNotification} = updateFields;

        if (addresses && addresses.length > 0) {
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

    async deleteWatcher(pushToken: string) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");

        const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }
        if(watcher.addresses!.length > 0) await this.explorerService.unsubscribeAddresses(watcher.addresses!);
        await this.watcherRepository.deleteWatcher({pushToken});


    }

    async deleteWatcherAddresses(pushToken: string, addresses: string[]) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");
        const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }
        if (addresses.length > 0) {
            await this.explorerService.unsubscribeAddresses(addresses);
            await this.watcherRepository.updateWatcher({pushToken}, {addresses: watcher.addresses?.filter(address => !addresses.includes(address))});
        }
    }

    async updateWatcherPushToken(oldPushToken: string, newPushToken: string, deviceId?: string) {
        const encodedOldPushToken = Buffer.from(oldPushToken).toString("base64");
        let watcher;
        if(deviceId) watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
        else watcher = await this.watcherRepository.getWatcher(encodedOldPushToken);
        if (!watcher) {
            throw new Error("watcher not found for device id or old push token");
        }


        await this.watcherRepository.updateWatcher({pushToken: oldPushToken}, {pushToken: newPushToken, deviceId});

    }
}
