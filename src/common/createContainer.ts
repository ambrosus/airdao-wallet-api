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

const fcmClient = admin.messaging();

export const createContainer = async (): Promise<DependencyContainer> => {

    const cacheStorage = new Redis(redisUrl);
    container.register<Redis>("Redis", { useValue: cacheStorage });

    const explorerService = container.resolve(ExplorerService);
    container.register<ExplorerService>(ExplorerService, { useValue: explorerService });
    
    const watcherRepository = container.resolve(WatcherRepository);
    container.register<WatcherRepository>(WatcherRepository, { useValue: watcherRepository });

    const notificationService = new NotificationService(fcmClient, androidChannel);
    container.register<NotificationService>(NotificationService, { useValue: notificationService });

    const watcherAddressesService = container.resolve(WatcherAddressesService);
    container.register<WatcherAddressesService>(WatcherAddressesService, { useValue: watcherAddressesService });

    const historicalNotificationsService = container.resolve(HistoricalNotificationsService);
    container.register<HistoricalNotificationsService>(HistoricalNotificationsService, { useValue: historicalNotificationsService });

    const cgPriceWatcher = new CgPriceWatcher(container.resolve("Redis"));
    container.register<CgPriceWatcher>(CgPriceWatcher, { useValue: cgPriceWatcher });

    const apiPriceWatcher = new ApiPriceWatcher(container.resolve("Redis"));
    container.register<ApiPriceWatcher>(ApiPriceWatcher, { useValue: apiPriceWatcher });

    const watcherService = new WatcherService(
        container.resolve("Redis"),
        container.resolve(ExplorerService),
        container.resolve(WatcherRepository),
        container.resolve(NotificationService),
        container.resolve(WatcherAddressesService),
        container.resolve(HistoricalNotificationsService)
    );
    container.register<WatcherService>(WatcherService, { useValue: watcherService });

    const priceWatcher = new PriceWatcher(
        container.resolve("Redis"),
        container.resolve(WatcherService),
        container.resolve(NotificationService),
        container.resolve(HistoricalNotificationsService)
    );

    container.register<PriceWatcher>(PriceWatcher, { useValue: priceWatcher });

    return container;
};