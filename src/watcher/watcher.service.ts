import "reflect-metadata";
import Redis from "ioredis";
import { singleton } from "tsyringe";

import { Watcher } from "./watcher.model";
import { ExplorerService } from "../explorer";
import { appEnv, explorerToken, notificationsTitleConfig, ONE_DAY_IN_MS, rewardsBankAddress } from "../config";
import { WatcherRepository } from "./watcher.repository";
import { NotificationService } from "../notification-sender";
import { WatcherAddressesService } from "../watcher-addresses";
import { HistoricalNotificationsService } from "../historical-notifications";
import { camelToSnake } from "../utils/camel-to-snake-case";
import { isERC20Standard } from "../utils/erc-standard-checker";

const SUPPORTED_TX_TYPES = ["Transfer", "TokenTransfer"];

@singleton()
export class WatcherService {

  constructor(
    private readonly cacheStorage: Redis,
    private readonly explorerService: ExplorerService,
    private readonly watcherRepository: WatcherRepository,
    private readonly notificationService: NotificationService,
    private readonly watcherAddressesService: WatcherAddressesService,
    private readonly historicalNotificationsService: HistoricalNotificationsService,
  ) {
  }

  async createWatcher(pushToken: string, deviceId?: string) {
    const encodedPushToken = Buffer.from(pushToken).toString("base64");

    if (deviceId) {
      const watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
      if (watcher) {
        const decodedPushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");
        await this.deleteWatcher(decodedPushToken);
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

  async getWatcher(pushToken: string) {
    const encodedPushToken = Buffer.from(pushToken).toString("base64");

    const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
    if (!watcher) throw new Error("watcher not found");

    const [addresses, historicalNotifications] = await Promise.all([
      this.watcherAddressesService.getWatcherAddressesWithDetails(watcher._id),
      this.historicalNotificationsService.getHistoricalNotifications(watcher._id)
    ]);

    const { _doc } = watcher as unknown as { _doc: Watcher };

    this.decodeWatcherToken(_doc);

    return camelToSnake({
      ..._doc,
      addresses,
      historicalNotifications
    });
  }

  async getAllWatchers(filter?: Record<string, string | unknown>) {
    return await this.watcherRepository.getAllWatchers(filter);
  }

  async getWatcherHistoryPrices() {
    return this.cacheStorage.get("cgPrices");
  }

  async updateWatcher(pushToken: string, updateFields: {
    addresses?: string[],
    threshold?: number,
    txNotification?: string,
    priceNotification?: string
  }) {
    const encodedPushToken = Buffer.from(pushToken).toString("base64");

    const watcher = await this.watcherRepository.getWatcher(encodedPushToken);

    if (!watcher) {
      throw new Error("watcher not found");
    }

    const {
      addresses,
      threshold,
      txNotification,
      priceNotification
    } = updateFields;

    const updates: { [key: string]: unknown } = {};

    if (addresses && addresses.length > 0) {
      const watcherAddresses = await this.watcherAddressesService.getWatcherAddresses(watcher._id);
      const uniqueAddresses = addresses.filter(address => !watcherAddresses.includes(address));

      if (uniqueAddresses.length > 0) {
        await Promise.all([
          this.explorerService.subscribeAddresses(uniqueAddresses),
          ...uniqueAddresses.map(address => this.watcherAddressesService.createWatcherAddress(watcher._id, address))
        ]);
      } else {
        throw new Error("400-No new addresses to add");
      }
    }

    if (threshold !== undefined) {
      updates.threshold = threshold;
    }
    if (txNotification !== undefined) {
      updates.txNotification = txNotification;
    }
    if (priceNotification !== undefined) {
      updates.priceNotification = priceNotification;
    }

    if (Object.keys(updates).length > 0) {
      await this.watcherRepository.updateWatcher({ pushToken: encodedPushToken }, updates);
    }
  }

  async updateWatcherPrice(pushToken: string, price: number) {
    await this.watcherRepository.updateWatcher({ pushToken }, { tokenPrice: price });
  }

  async updateWatcherPushToken(oldPushToken: string, newPushToken: string, deviceId?: string) {
    const encodedOldPushToken = Buffer.from(oldPushToken).toString("base64");
    const encodedNewPushToken = Buffer.from(newPushToken).toString("base64");

    let watcher;
    if (deviceId) watcher = await this.watcherRepository.getWatcherByDeviceId(deviceId);
    else watcher = await this.watcherRepository.getWatcher(encodedOldPushToken);

    if (!watcher) {
      throw new Error("watcher not found for device id or old push token");
    }

    await this.watcherRepository.updateWatcher({ pushToken: encodedOldPushToken }, {
      pushToken: encodedNewPushToken,
      deviceId
    });
  }

  async deleteWatcher(pushToken: string) {
    const encodedPushToken = Buffer.from(pushToken).toString("base64");

    const watcher = await this.watcherRepository.getWatcher(encodedPushToken);
    if (!watcher) {
      throw new Error("watcher not found");
    }

    const watcherAddresses = await this.watcherAddressesService.getWatcherAddresses(watcher._id);
    if (watcherAddresses.length > 0) await this.explorerService.unsubscribeAddresses(watcherAddresses);

    await Promise.all([
      this.watcherAddressesService.deleteAllWatcherAddresses(watcher._id),
      this.watcherRepository.deleteWatcher({ pushToken: encodedPushToken })
    ]);

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
      await Promise.all([
        this.explorerService.unsubscribeAddresses(allowedAddresses),
        this.watcherAddressesService.deleteWatcherAddresses(watcher._id, allowedAddresses)
      ]);
    }
  }

  async deleteWatchersWithStaleData() {
    const watchers = await this.watcherRepository.getAllWatchers({
      lastSuccessDate: {
        $lt: new Date(Date.now() - 7 * ONE_DAY_IN_MS)
      }
    });

    if (watchers.length === 0) return;

    const deletions = [];
    for (const watcher of watchers) {
      deletions.push(
        Promise.all([
          this.watcherRepository.deleteWatcher({ _id: watcher._id }),
          this.watcherAddressesService.deleteAllWatcherAddresses(watcher._id)
        ])
      );
    }
    await Promise.all(deletions);
  }

  async watcherCallback(id: string, items: { address: string, txHash: string }[]) {
    if (id !== explorerToken) throw new Error("Wrong Explorer API ID");
    const promises = items.map(async (item) => {
      await this.handleCallback(item.address, item.txHash);
    });

    await Promise.all(promises);
  }

  async handleCallback(address: string, txHash: string) {
    console.log("Processing explorer callback");
    const watcherIds = await this.watcherAddressesService.getWatcherIdsByAddress(address);
    if (watcherIds.length === 0) return;

    const watchers = await this.watcherRepository.getAllWatchers({
      _id: { $in: watcherIds },
      txNotification: { $regex: /^on$/i }
    });
    if (watchers.length === 0) return;

    const txDataResponse = await this.explorerService.getTransactionData(txHash);
    if (!txDataResponse || txDataResponse.data.data.length === 0) return;

    const { data: { data: [txData] } } = txDataResponse;
    const { from, to, value, timestamp, token, type } = txData;
    console.log("Tx Data: ", from, to, type);

    if (!SUPPORTED_TX_TYPES.includes(type)) return;
    console.log("After check Tx Data: ", from, to, type);

    // @dev To avoid sending notifications for node rewards transactions
    // @dev To lower case for case-insensitive comparison
    if (from.toLowerCase() === rewardsBankAddress.toLowerCase()) return;
    console.log("After check address ", from, to, type);

    // @dev Did it for hiding ERC-1155 and ERC-721 transfers for users
    if (token && !(await isERC20Standard(token.address))) {
      return;
    }

    const cutAddress = (addr: string) => addr ? `${addr.slice(0, 5)}...${addr.slice(-5)}` : "";

    const roundedAmount = Number(value.ether).toFixed(2);
    const tokenSymbol = value.symbol ?? (token ? token.symbol : "AMB");

    const data = {
      type: "transaction-alert",
      timestamp,
      sender: cutAddress(from),
      to: cutAddress(to),
      amount: roundedAmount,
      symbol: tokenSymbol
    };


    const notifications = watchers.map(async (watcher) => {
      const decodedPushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");

      const title = notificationsTitleConfig[appEnv].txAlert;
      const body = `From: ${cutAddress(from)}\nTo: ${cutAddress(to)}\nAmount: ${roundedAmount} ${tokenSymbol}`;

      try {
        await this.notificationService.sendNotification({ title, body, pushToken: decodedPushToken, data });
        await this.watcherRepository.updateWatcher({ _id: watcher._id }, { lastSuccessDate: Date.now() });
      } catch (error) {
        if ((error as Error).message.includes("messaging/registration-token-not-registered")) {
          await this.watcherRepository.updateWatcher({ _id: watcher._id }, { lastFailDate: Date.now() });
        }
      }

      await Promise.all([
        this.historicalNotificationsService.addHistoricalNotification(watcher._id, {
          title,
          body,
          sent: true,
          timestamp: Date.now()
        }),
        this.watcherAddressesService.updateWatcherAddress(watcher._id, address, { txHash })
      ]);
    });

    await Promise.all(notifications);
  }

  async subscribeToExplorer() {
    console.log("Subscribing to explorer...");
    const addresses = await this.watcherAddressesService.getUniqueAddresses();

    if (addresses.length === 0) return;

    await this.explorerService.subscribeAddresses(addresses);
  }

  private decodeWatcherToken(watcher: Watcher) {
    watcher.pushToken = Buffer.from(watcher.pushToken, "base64").toString("utf-8");
  }
}
