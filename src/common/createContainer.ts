import "reflect-metadata";
import Redis from "ioredis";
import * as admin from "firebase-admin";
import { container, DependencyContainer } from "tsyringe";

import { ExplorerService } from "../explorer";
import { NotificationService } from "../notification-sender";
import { WatcherRepository, WatcherService } from "../watcher";
import { WatcherAddressesService } from "../watcher-addresses";
import { androidChannel, firebaseCredPath, redisUrl } from "../config";
import { HistoricalNotificationsService } from "../historical-notifications";
import { ApiPriceWatcher, CgPriceWatcher, PriceWatcher } from "../price-watchers";

// eslint-disable-next-line @typescript-eslint/no-var-requires
const serviceAccount = require(firebaseCredPath);

admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});
console.log("Firebase initialized");

const fcmClient = admin.messaging();

export const createContainer = async (): Promise<DependencyContainer> => {
  console.log("Starting to create container");

  const cacheStorage = new Redis(redisUrl);
  console.log("Redis initialized");
  container.register<Redis>("Redis", { useValue: cacheStorage });
  console.log("Redis registered in container");

  const explorerService = container.resolve(ExplorerService);
  console.log("ExplorerService resolved");
  container.register<ExplorerService>(ExplorerService, { useValue: explorerService });
  console.log("ExplorerService registered in container");

  const watcherRepository = container.resolve(WatcherRepository);
  console.log("WatcherRepository resolved");
  container.register<WatcherRepository>(WatcherRepository, { useValue: watcherRepository });
  console.log("WatcherRepository registered in container");

  const notificationService = new NotificationService(fcmClient, androidChannel);
  console.log("NotificationService created");
  container.register<NotificationService>(NotificationService, { useValue: notificationService });
  console.log("NotificationService registered in container");

  const watcherAddressesService = container.resolve(WatcherAddressesService);
  console.log("WatcherAddressesService resolved");
  container.register<WatcherAddressesService>(WatcherAddressesService, { useValue: watcherAddressesService });
  console.log("WatcherAddressesService registered in container");

  const historicalNotificationsService = container.resolve(HistoricalNotificationsService);
  console.log("HistoricalNotificationsService resolved");
  container.register<HistoricalNotificationsService>(HistoricalNotificationsService, { useValue: historicalNotificationsService });
  console.log("HistoricalNotificationsService registered in container");

  const cgPriceWatcher = new CgPriceWatcher(container.resolve("Redis"));
  console.log("CgPriceWatcher created");
  container.register<CgPriceWatcher>(CgPriceWatcher, { useValue: cgPriceWatcher });
  console.log("CgPriceWatcher registered in container");

  const apiPriceWatcher = new ApiPriceWatcher(container.resolve("Redis"));
  console.log("ApiPriceWatcher created");
  container.register<ApiPriceWatcher>(ApiPriceWatcher, { useValue: apiPriceWatcher });
  console.log("ApiPriceWatcher registered in container");

  const watcherService = new WatcherService(
    container.resolve("Redis"),
    container.resolve(ExplorerService),
    container.resolve(WatcherRepository),
    container.resolve(NotificationService),
    container.resolve(WatcherAddressesService),
    container.resolve(HistoricalNotificationsService)
  );
  console.log("WatcherService created");
  container.register<WatcherService>(WatcherService, { useValue: watcherService });
  console.log("WatcherService registered in container");

  const priceWatcher = new PriceWatcher(
    container.resolve("Redis"),
    container.resolve(WatcherService),
    container.resolve(NotificationService),
    container.resolve(HistoricalNotificationsService)
  );
  console.log("PriceWatcher created");
  container.register<PriceWatcher>(PriceWatcher, { useValue: priceWatcher });
  console.log("PriceWatcher registered in container");

  console.log("Container creation completed");
  return container;
};