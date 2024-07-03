import Redis from "ioredis";
import { singleton } from "tsyringe";
import { Watcher } from "./watcher.model";
import { ExplorerService } from "../explorer";
import { WatcherRepository } from "./watcher.repository";
import { NotificationService } from "../notification-sender";
import { WatcherAddressesService } from "../watcher-addresses";
import { HistoricalNotificationsService } from "../historical-notifications";
import {explorerToken} from "../config";

@singleton()
export class WatcherService {

    constructor(
        private readonly cacheStorage: Redis,
        private readonly explorerService: ExplorerService,
        private readonly watcherRepository: WatcherRepository,
        private readonly notificationService: NotificationService,
        private readonly watcherAddressesService: WatcherAddressesService,
        private readonly historicalNotificationsService: HistoricalNotificationsService,
    ) {}

    async getWatcher(pushToken: string) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");

        const watcher =  await this.watcherRepository.getWatcher(encodedPushToken);
        if (!watcher) throw new Error("watcher not found");

        const watcherAddresses = await this.watcherAddressesService.getWatcherAddressesWithDetails(watcher._id);
        const historicalNotifications = await this.historicalNotificationsService.getHistoricalNotifications(watcher._id);

        return {
            ...watcher,
            addresses: watcherAddresses,
            historicalNotifications
        };
    }

    async getAllWatchers(filter?: Record<string, string>) {
        return await this.watcherRepository.getAllWatchers(filter);
    }

    async getWatcherHistoryPrices() {
        return this.cacheStorage.get("cgPrices");
    }

    async createWatcher(pushToken: string, deviceId?: string) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");

        if (deviceId) {
            const watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
            if (watcher) {
                await this.watcherRepository.deleteWatcher({ deviceId });
            }
        }

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
            const watcherAddresses = await this.watcherAddressesService.getWatcherAddresses(watcher._id);
            const uniqueAddresses = addresses.filter(address => watcherAddresses.includes(address)) || [];

            if (uniqueAddresses.length > 0) {
                await this.explorerService.subscribeAddresses(uniqueAddresses);
                await this.watcherRepository.updateWatcher({pushToken}, {addresses: uniqueAddresses});
            }
            else throw new Error("400-No new addresses to add");
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

        const watcherAddresses = await this.watcherAddressesService.getWatcherAddresses(watcher._id);
        if (watcherAddresses.length > 0) await this.explorerService.unsubscribeAddresses(watcherAddresses);

        await this.watcherAddressesService.deleteAllWatcherAddresses(watcher._id);
        await this.watcherRepository.deleteWatcher({pushToken});
    }

    async deleteWatcherAddresses(pushToken: string, addresses: string[]) {
        const encodedPushToken = Buffer.from(pushToken).toString("base64");

        const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
        if (!watcher) {
            throw new Error("watcher not found");
        }

        const watcherAddresses = await this.watcherAddressesService.getWatcherAddresses(watcher._id);
        // To avoid deleting redundant addresses
        const allowedAddresses = watcherAddresses.filter(address => addresses.includes(address));

        if (allowedAddresses.length > 0) {
            await this.explorerService.unsubscribeAddresses(allowedAddresses);
            await this.watcherAddressesService.deleteWatcherAddresses(watcher._id, allowedAddresses);
        }
    }

    async updateWatcherPushToken(oldPushToken: string, newPushToken: string, deviceId?: string) {
        const encodedOldPushToken = Buffer.from(oldPushToken).toString("base64");

        let watcher;
        if (deviceId) watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
        else watcher = await this.watcherRepository.getWatcher(encodedOldPushToken);

        if (!watcher) {
            throw new Error("watcher not found for device id or old push token");
        }

        await this.watcherRepository.updateWatcher({pushToken: oldPushToken}, {pushToken: newPushToken, deviceId});
    }

    async watcherCallback(id: string, items: { address: string, txHash: string }[]) {
        if (id !== explorerToken) throw new Error("Wrong Explorer API ID");
        const promises = items.map(async (item) => {
            await this.handleCallback(item.address, item.txHash);
        });

        await Promise.all(promises);
    }

    async handleCallback(address: string, txHash: string) {
        const watcherIds = await this.watcherAddressesService.getWatcherIdsByAddress(address);

        const watchers = await this.watcherRepository.getAllWatchers({
            _id: { $in: watcherIds },
            txNotification: "ON"
        });
        let cutFromAddress = "";
        let cutToAddress = "";
        let tokenSymbol = "";


        const txDataResponse = await this.explorerService.getTransactionData(txHash) ;
        if (!txDataResponse || txDataResponse.data.data.length === 0) return;
        const { data: { data: [txData] } } = txDataResponse;

        if (txData.from.length > 0 && txData.from !== "") {
             cutFromAddress = `${txData.from.slice(0, 5)}...${txData.from.slice(-5)}`;
        }

        if (txData.to.length > 0 && txData.to !== "") {
             cutToAddress = `${txData.to.slice(0, 5)}...${txData.to.slice(-5)}`;
        }

        const roundedAmount = Number(txData.value.ether).toFixed(2);

        if (!txData.value.symbol) {
            tokenSymbol = "AMB";
        } else if (txData.value.symbol === "") {
            tokenSymbol = "HPT";
        }
        else {
            tokenSymbol = txData.value.symbol;
        }
        const data = {
            type: "transaction-alert",
            timestamp: txData.timestamp,
            sender: cutFromAddress,
            to: cutToAddress,
            amount: roundedAmount,
            symbol: tokenSymbol
        };

        for (const watcher of watchers) {
            const decodedPushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");
            const title = "AMB-Net Tx Alert";
            const body = `From: ${ cutFromAddress }\nTo: ${ cutToAddress }\nAmount: ${ roundedAmount } ${ tokenSymbol }`;

            try {
                await this.notificationService.sendNotification({ title, body, pushToken: decodedPushToken, data });
            } catch (error) {
                if ((error as Error).message === "http error status: 404; reason: app instance has been unregistered; code: registration-token-not-registered; details: Requested entity was not found.") {
                    await this.watcherRepository.updateWatcher({ _id: watcher._id }, { lastFailDate: Date.now() });
                }
            }
        }

    }
}
