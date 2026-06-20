export namespace app {
	
	export class Account {
	    id: number;
	    uid: string;
	    fullData: string;
	    password: string;
	    twofa: string;
	    email: string;
	    passMail: string;
	    mailRecovery: string;
	    cookie: string;
	    token: string;
	    status: string;
	    checkpoint: string;
	    statusAds: string;
	    bm: string;
	    tkqc: string;
	    chatSupport: string;
	    fullName: string;
	    location: string;
	    avatar: string;
	    cover: string;
	    phone: string;
	    proxy: string;
	    userAgent: string;
	    note: string;
	    noteRun: string;
	    importTime: string;
	    category: string;
	    lastRun: string;
	    activity: string;
	    sourceCode: string;
	    categoryId?: number;
	    emailMeta?: string;
	    srnonce?: string;
	    sessionlessCryptedUID?: string;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.uid = source["uid"];
	        this.fullData = source["fullData"];
	        this.password = source["password"];
	        this.twofa = source["twofa"];
	        this.email = source["email"];
	        this.passMail = source["passMail"];
	        this.mailRecovery = source["mailRecovery"];
	        this.cookie = source["cookie"];
	        this.token = source["token"];
	        this.status = source["status"];
	        this.checkpoint = source["checkpoint"];
	        this.statusAds = source["statusAds"];
	        this.bm = source["bm"];
	        this.tkqc = source["tkqc"];
	        this.chatSupport = source["chatSupport"];
	        this.fullName = source["fullName"];
	        this.location = source["location"];
	        this.avatar = source["avatar"];
	        this.cover = source["cover"];
	        this.phone = source["phone"];
	        this.proxy = source["proxy"];
	        this.userAgent = source["userAgent"];
	        this.note = source["note"];
	        this.noteRun = source["noteRun"];
	        this.importTime = source["importTime"];
	        this.category = source["category"];
	        this.lastRun = source["lastRun"];
	        this.activity = source["activity"];
	        this.sourceCode = source["sourceCode"];
	        this.categoryId = source["categoryId"];
	        this.emailMeta = source["emailMeta"];
	        this.srnonce = source["srnonce"];
	        this.sessionlessCryptedUID = source["sessionlessCryptedUID"];
	    }
	}
	export class AccountFilter {
	    keyword: string;
	    status: string;
	    categoryId?: number;
	    sortBy: string;
	    sortDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.status = source["status"];
	        this.categoryId = source["categoryId"];
	        this.sortBy = source["sortBy"];
	        this.sortDir = source["sortDir"];
	    }
	}
	export class AccountListResult {
	    items: Account[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Account);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppResourceUsage {
	    ramMb: number;
	    cpuPct: number;
	
	    static createFrom(source: any = {}) {
	        return new AppResourceUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ramMb = source["ramMb"];
	        this.cpuPct = source["cpuPct"];
	    }
	}
	export class CloneHVStockResult {
	    name: string;
	    amount: string;
	    price: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CloneHVStockResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.amount = source["amount"];
	        this.price = source["price"];
	        this.error = source["error"];
	    }
	}
	export class DeleteResult {
	    deleted: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deleted = source["deleted"];
	    }
	}
	export class FbAppStatus {
	    path: string;
	    count: number;
	    overrideActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FbAppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.count = source["count"];
	        this.overrideActive = source["overrideActive"];
	    }
	}
	export class GeneralConfig {
	    threadRequest: number;
	    delayRequest: number;
	    delayThread: number;
	    apiCheckIp: number;
	    threadCheckInfo: number;
	    loginPlatform: string;
	    loginMethod: number;
	    saveRunColumn: boolean;
	    backupDB: boolean;
	    closeAfterDone: boolean;
	    accountSourcePath: string;
	    accountSource: string;
	    cloneHvUsername: string;
	    cloneHvPassword: string;
	    cloneHvProductId: string;
	    cloneHvAmount: number;
	    captchaProvider: string;
	    captchaKeys: Record<string, string>;
	    ipProvider: string;
	    checkIpBeforeRun: boolean;
	    delayChangeIp: number;
	    localeFake: string;
	    deepFakeInApi: boolean;
	    cookieUse: boolean;
	    cookieLimit: boolean;
	    cookieLimitCount: number;
	    cookieMode: string;
	    uaAddSpecs: boolean;
	    uaBuildFile: boolean;
	    uaCustomType: number;
	    simNetworkMode: string;
	    simNetworkType: string;
	
	    static createFrom(source: any = {}) {
	        return new GeneralConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.threadRequest = source["threadRequest"];
	        this.delayRequest = source["delayRequest"];
	        this.delayThread = source["delayThread"];
	        this.apiCheckIp = source["apiCheckIp"];
	        this.threadCheckInfo = source["threadCheckInfo"];
	        this.loginPlatform = source["loginPlatform"];
	        this.loginMethod = source["loginMethod"];
	        this.saveRunColumn = source["saveRunColumn"];
	        this.backupDB = source["backupDB"];
	        this.closeAfterDone = source["closeAfterDone"];
	        this.accountSourcePath = source["accountSourcePath"];
	        this.accountSource = source["accountSource"];
	        this.cloneHvUsername = source["cloneHvUsername"];
	        this.cloneHvPassword = source["cloneHvPassword"];
	        this.cloneHvProductId = source["cloneHvProductId"];
	        this.cloneHvAmount = source["cloneHvAmount"];
	        this.captchaProvider = source["captchaProvider"];
	        this.captchaKeys = source["captchaKeys"];
	        this.ipProvider = source["ipProvider"];
	        this.checkIpBeforeRun = source["checkIpBeforeRun"];
	        this.delayChangeIp = source["delayChangeIp"];
	        this.localeFake = source["localeFake"];
	        this.deepFakeInApi = source["deepFakeInApi"];
	        this.cookieUse = source["cookieUse"];
	        this.cookieLimit = source["cookieLimit"];
	        this.cookieLimitCount = source["cookieLimitCount"];
	        this.cookieMode = source["cookieMode"];
	        this.uaAddSpecs = source["uaAddSpecs"];
	        this.uaBuildFile = source["uaBuildFile"];
	        this.uaCustomType = source["uaCustomType"];
	        this.simNetworkMode = source["simNetworkMode"];
	        this.simNetworkType = source["simNetworkType"];
	    }
	}
	export class ImportResult {
	    imported: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.errors = source["errors"];
	    }
	}
	export class PlatformUAConfig {
	    useOriginalUA: boolean;
	    addVirtualSpecAndroid: boolean;
	    buildUA: boolean;
	    replaceCarrier: boolean;
	    trackingID: boolean;
	    uaPoolKey: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformUAConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.useOriginalUA = source["useOriginalUA"];
	        this.addVirtualSpecAndroid = source["addVirtualSpecAndroid"];
	        this.buildUA = source["buildUA"];
	        this.replaceCarrier = source["replaceCarrier"];
	        this.trackingID = source["trackingID"];
	        this.uaPoolKey = source["uaPoolKey"];
	        this.kind = source["kind"];
	    }
	}
	export class InteractionConfig {
	    verifyEnabled: boolean;
	    mailProvider: string;
	    mailList: string;
	    checkLiveDieEnabled: boolean;
	    timeDelayCheck: number;
	    timeDelaySendCode: number;
	    sendAgainCode: boolean;
	    outputPath: string;
	    uaPoolKey?: string;
	    zeusXApiKey: string;
	    zeusXAccountCode: string;
	    dvfbApiKey: string;
	    dvfbAccountType: string;
	    store1sApiKey: string;
	    store1sProductId: string;
	    mail30sApiKey: string;
	    mail30sProductSlug: string;
	    tempMailLolApiKey: string;
	    tempMailDomain: string;
	    tempMailDomains?: Record<string, string>;
	    tempMailToken?: string;
	    tempMailTokens?: Record<string, string>;
	    muaMailApiKey: string;
	    muaMailProductId: string;
	    unlimitMailApiKey: string;
	    unlimitMailProductId: string;
	    sptMailApiKey: string;
	    sptMailServiceCode: string;
	    emailAPIInfoApiKey: string;
	    emailAPIInfoProductCode: string;
	    otpCheapApiKey: string;
	    otpCheapServiceId: string;
	    shopGmail9999ApiKey: string;
	    shopGmail9999Service: string;
	    rentGmailApiKey: string;
	    rentGmailPlatform: string;
	    otpCodesSmsApiKey: string;
	    otpCodesSmsServiceId: string;
	    wmemailApiKey: string;
	    wmemailCommodity: string;
	    priyoEmailApiKey: string;
	    otpHotmailPriority: string;
	    mailPoolBatch: number;
	    waitCode: number;
	    waitMail: number;
	    trySendCode: number;
	    useMailTimes: number;
	    delayConfirmEmail: number;
	    delayCheckLive: number;
	    delayVeriReg: number;
	    addMailRetry: number;
	    retryUnknownNow: boolean;
	    apiVerifyPlatform: string;
	    apiVerifyPlatforms?: string[];
	    apiVerifyTokenType: string;
	    apiRegPlatform: string;
	    apiRegPlatforms?: string[];
	    delayReg: number;
	    delayStep: number;
	    leadDomainMail: string;
	    passwordReg: string;
	    nameRegLocale: string;
	    regMode: string;
	    regModeRotate: boolean;
	    regModeRotateMailMinutes: number;
	    regModeRotatePhoneMinutes: number;
	    verifyAfterReg: boolean;
	    phoneMailMode: string;
	    fmPhoneCode: boolean;
	    useUGForVerify: boolean;
	    regForVerify: boolean;
	    cookieInitialMethod: string;
	    limitCookieInitial: boolean;
	    limitCookieInitialCount: number;
	    cookieInitialFile: string;
	    limitCheckpoint: boolean;
	    limitCheckpointCount: number;
	    deleteDatrCheckpoint: boolean;
	    limitDatrAge: boolean;
	    limitDatrAgeMinutes: number;
	    saveNewDatr: boolean;
	    createEnabled: boolean;
	    createType: string;
	    createCookieList: string;
	    createOutputPath: string;
	    resultFolderPath: string;
	    splitMode: boolean;
	    splitVerifyThreads: number;
	    regThreads: number;
	    autoRestartEnabled: boolean;
	    autoRestartMinutes: number;
	    verifySourceFolderPath: string;
	    keepIpSuccess: boolean;
	    keepUaSuccess: boolean;
	    keepDatrSuccess: boolean;
	    keepInitialSuccess: boolean;
	    addVirtualSpecAndroid: boolean;
	    buildUA: boolean;
	    useOriginalUA: boolean;
	    replaceCarrier: boolean;
	    trackingIDReg: boolean;
	    trackingIDVer: boolean;
	    regPlatformUA?: Record<string, PlatformUAConfig>;
	    verifyPlatformUA?: Record<string, PlatformUAConfig>;
	    reUseEmail: boolean;
	    useEmailTime: number;
	    fmUserTmpMail: boolean;
	    useProxyTempmail: boolean;
	    useProxyGmail: boolean;
	    enable2fa: boolean;
	    getNewDatrOnLive: boolean;
	    uploadAvatar: boolean;
	    avatarFolderPath: string;
	    delayDisplayResult: number;
	    addInfo: boolean;
	    addInfoCity: boolean;
	    addInfoHometown: boolean;
	    addInfoSchool: boolean;
	    addInfoCollege: boolean;
	    addInfoWork: boolean;
	    addInfoRelationship: boolean;
	    addInfoDataDir: string;
	    addInfoDelayMs: number;
	    autoUploadAfterReg: boolean;
	    autoUploadAfterVerify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InteractionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.verifyEnabled = source["verifyEnabled"];
	        this.mailProvider = source["mailProvider"];
	        this.mailList = source["mailList"];
	        this.checkLiveDieEnabled = source["checkLiveDieEnabled"];
	        this.timeDelayCheck = source["timeDelayCheck"];
	        this.timeDelaySendCode = source["timeDelaySendCode"];
	        this.sendAgainCode = source["sendAgainCode"];
	        this.outputPath = source["outputPath"];
	        this.uaPoolKey = source["uaPoolKey"];
	        this.zeusXApiKey = source["zeusXApiKey"];
	        this.zeusXAccountCode = source["zeusXAccountCode"];
	        this.dvfbApiKey = source["dvfbApiKey"];
	        this.dvfbAccountType = source["dvfbAccountType"];
	        this.store1sApiKey = source["store1sApiKey"];
	        this.store1sProductId = source["store1sProductId"];
	        this.mail30sApiKey = source["mail30sApiKey"];
	        this.mail30sProductSlug = source["mail30sProductSlug"];
	        this.tempMailLolApiKey = source["tempMailLolApiKey"];
	        this.tempMailDomain = source["tempMailDomain"];
	        this.tempMailDomains = source["tempMailDomains"];
	        this.tempMailToken = source["tempMailToken"];
	        this.tempMailTokens = source["tempMailTokens"];
	        this.muaMailApiKey = source["muaMailApiKey"];
	        this.muaMailProductId = source["muaMailProductId"];
	        this.unlimitMailApiKey = source["unlimitMailApiKey"];
	        this.unlimitMailProductId = source["unlimitMailProductId"];
	        this.sptMailApiKey = source["sptMailApiKey"];
	        this.sptMailServiceCode = source["sptMailServiceCode"];
	        this.emailAPIInfoApiKey = source["emailAPIInfoApiKey"];
	        this.emailAPIInfoProductCode = source["emailAPIInfoProductCode"];
	        this.otpCheapApiKey = source["otpCheapApiKey"];
	        this.otpCheapServiceId = source["otpCheapServiceId"];
	        this.shopGmail9999ApiKey = source["shopGmail9999ApiKey"];
	        this.shopGmail9999Service = source["shopGmail9999Service"];
	        this.rentGmailApiKey = source["rentGmailApiKey"];
	        this.rentGmailPlatform = source["rentGmailPlatform"];
	        this.otpCodesSmsApiKey = source["otpCodesSmsApiKey"];
	        this.otpCodesSmsServiceId = source["otpCodesSmsServiceId"];
	        this.wmemailApiKey = source["wmemailApiKey"];
	        this.wmemailCommodity = source["wmemailCommodity"];
	        this.priyoEmailApiKey = source["priyoEmailApiKey"];
	        this.otpHotmailPriority = source["otpHotmailPriority"];
	        this.mailPoolBatch = source["mailPoolBatch"];
	        this.waitCode = source["waitCode"];
	        this.waitMail = source["waitMail"];
	        this.trySendCode = source["trySendCode"];
	        this.useMailTimes = source["useMailTimes"];
	        this.delayConfirmEmail = source["delayConfirmEmail"];
	        this.delayCheckLive = source["delayCheckLive"];
	        this.delayVeriReg = source["delayVeriReg"];
	        this.addMailRetry = source["addMailRetry"];
	        this.retryUnknownNow = source["retryUnknownNow"];
	        this.apiVerifyPlatform = source["apiVerifyPlatform"];
	        this.apiVerifyPlatforms = source["apiVerifyPlatforms"];
	        this.apiVerifyTokenType = source["apiVerifyTokenType"];
	        this.apiRegPlatform = source["apiRegPlatform"];
	        this.apiRegPlatforms = source["apiRegPlatforms"];
	        this.delayReg = source["delayReg"];
	        this.delayStep = source["delayStep"];
	        this.leadDomainMail = source["leadDomainMail"];
	        this.passwordReg = source["passwordReg"];
	        this.nameRegLocale = source["nameRegLocale"];
	        this.regMode = source["regMode"];
	        this.regModeRotate = source["regModeRotate"];
	        this.regModeRotateMailMinutes = source["regModeRotateMailMinutes"];
	        this.regModeRotatePhoneMinutes = source["regModeRotatePhoneMinutes"];
	        this.verifyAfterReg = source["verifyAfterReg"];
	        this.phoneMailMode = source["phoneMailMode"];
	        this.fmPhoneCode = source["fmPhoneCode"];
	        this.useUGForVerify = source["useUGForVerify"];
	        this.regForVerify = source["regForVerify"];
	        this.cookieInitialMethod = source["cookieInitialMethod"];
	        this.limitCookieInitial = source["limitCookieInitial"];
	        this.limitCookieInitialCount = source["limitCookieInitialCount"];
	        this.cookieInitialFile = source["cookieInitialFile"];
	        this.limitCheckpoint = source["limitCheckpoint"];
	        this.limitCheckpointCount = source["limitCheckpointCount"];
	        this.deleteDatrCheckpoint = source["deleteDatrCheckpoint"];
	        this.limitDatrAge = source["limitDatrAge"];
	        this.limitDatrAgeMinutes = source["limitDatrAgeMinutes"];
	        this.saveNewDatr = source["saveNewDatr"];
	        this.createEnabled = source["createEnabled"];
	        this.createType = source["createType"];
	        this.createCookieList = source["createCookieList"];
	        this.createOutputPath = source["createOutputPath"];
	        this.resultFolderPath = source["resultFolderPath"];
	        this.splitMode = source["splitMode"];
	        this.splitVerifyThreads = source["splitVerifyThreads"];
	        this.regThreads = source["regThreads"];
	        this.autoRestartEnabled = source["autoRestartEnabled"];
	        this.autoRestartMinutes = source["autoRestartMinutes"];
	        this.verifySourceFolderPath = source["verifySourceFolderPath"];
	        this.keepIpSuccess = source["keepIpSuccess"];
	        this.keepUaSuccess = source["keepUaSuccess"];
	        this.keepDatrSuccess = source["keepDatrSuccess"];
	        this.keepInitialSuccess = source["keepInitialSuccess"];
	        this.addVirtualSpecAndroid = source["addVirtualSpecAndroid"];
	        this.buildUA = source["buildUA"];
	        this.useOriginalUA = source["useOriginalUA"];
	        this.replaceCarrier = source["replaceCarrier"];
	        this.trackingIDReg = source["trackingIDReg"];
	        this.trackingIDVer = source["trackingIDVer"];
	        this.regPlatformUA = this.convertValues(source["regPlatformUA"], PlatformUAConfig, true);
	        this.verifyPlatformUA = this.convertValues(source["verifyPlatformUA"], PlatformUAConfig, true);
	        this.reUseEmail = source["reUseEmail"];
	        this.useEmailTime = source["useEmailTime"];
	        this.fmUserTmpMail = source["fmUserTmpMail"];
	        this.useProxyTempmail = source["useProxyTempmail"];
	        this.useProxyGmail = source["useProxyGmail"];
	        this.enable2fa = source["enable2fa"];
	        this.getNewDatrOnLive = source["getNewDatrOnLive"];
	        this.uploadAvatar = source["uploadAvatar"];
	        this.avatarFolderPath = source["avatarFolderPath"];
	        this.delayDisplayResult = source["delayDisplayResult"];
	        this.addInfo = source["addInfo"];
	        this.addInfoCity = source["addInfoCity"];
	        this.addInfoHometown = source["addInfoHometown"];
	        this.addInfoSchool = source["addInfoSchool"];
	        this.addInfoCollege = source["addInfoCollege"];
	        this.addInfoWork = source["addInfoWork"];
	        this.addInfoRelationship = source["addInfoRelationship"];
	        this.addInfoDataDir = source["addInfoDataDir"];
	        this.addInfoDelayMs = source["addInfoDelayMs"];
	        this.autoUploadAfterReg = source["autoUploadAfterReg"];
	        this.autoUploadAfterVerify = source["autoUploadAfterVerify"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IpConfig {
	    proxyList: string;
	    proxyStickyList: string;
	    proxyActiveTab: string;
	    proxyType: string;
	    fptKeys: string;
	    xproxyServiceUrl: string;
	    xproxyType: string;
	    xproxyList: string;
	    xproxyThreadPerIp: number;
	    xproxyRunType: string;
	    tinsoftKeys: string;
	    tinsoftThreadPerIp: number;
	    shoplikeKeys: string;
	    shoplikeThreadPerIp: number;
	    netproxyKeys: string;
	    netproxyThreadPerIp: number;
	    minproxyKeys: string;
	    minproxyThreadPerIp: number;
	    netproxyGbKey: string;
	    proxyPopularKeys: string;
	    proxyPopularThreadPerIp: number;
	    proxyPopularAccessToken: string;
	    proxyFarmKeys: string;
	    proxyFarmThreadPerIp: number;
	    proxyFarmAccessToken: string;
	    useVerifyProxyForReg: boolean;
	    regIpProvider: string;
	    regProxyList: string;
	    regProxyStickyList: string;
	    regProxyActiveTab: string;
	    regProxyType: string;
	    proxyRetry: number;
	    proxyDelayMs: number;
	
	    static createFrom(source: any = {}) {
	        return new IpConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyList = source["proxyList"];
	        this.proxyStickyList = source["proxyStickyList"];
	        this.proxyActiveTab = source["proxyActiveTab"];
	        this.proxyType = source["proxyType"];
	        this.fptKeys = source["fptKeys"];
	        this.xproxyServiceUrl = source["xproxyServiceUrl"];
	        this.xproxyType = source["xproxyType"];
	        this.xproxyList = source["xproxyList"];
	        this.xproxyThreadPerIp = source["xproxyThreadPerIp"];
	        this.xproxyRunType = source["xproxyRunType"];
	        this.tinsoftKeys = source["tinsoftKeys"];
	        this.tinsoftThreadPerIp = source["tinsoftThreadPerIp"];
	        this.shoplikeKeys = source["shoplikeKeys"];
	        this.shoplikeThreadPerIp = source["shoplikeThreadPerIp"];
	        this.netproxyKeys = source["netproxyKeys"];
	        this.netproxyThreadPerIp = source["netproxyThreadPerIp"];
	        this.minproxyKeys = source["minproxyKeys"];
	        this.minproxyThreadPerIp = source["minproxyThreadPerIp"];
	        this.netproxyGbKey = source["netproxyGbKey"];
	        this.proxyPopularKeys = source["proxyPopularKeys"];
	        this.proxyPopularThreadPerIp = source["proxyPopularThreadPerIp"];
	        this.proxyPopularAccessToken = source["proxyPopularAccessToken"];
	        this.proxyFarmKeys = source["proxyFarmKeys"];
	        this.proxyFarmThreadPerIp = source["proxyFarmThreadPerIp"];
	        this.proxyFarmAccessToken = source["proxyFarmAccessToken"];
	        this.useVerifyProxyForReg = source["useVerifyProxyForReg"];
	        this.regIpProvider = source["regIpProvider"];
	        this.regProxyList = source["regProxyList"];
	        this.regProxyStickyList = source["regProxyStickyList"];
	        this.regProxyActiveTab = source["regProxyActiveTab"];
	        this.regProxyType = source["regProxyType"];
	        this.proxyRetry = source["proxyRetry"];
	        this.proxyDelayMs = source["proxyDelayMs"];
	    }
	}
	export class LegacyFieldEntry {
	    legacyKey: string;
	    newPath: string;
	    displayValue: string;
	    status: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new LegacyFieldEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.legacyKey = source["legacyKey"];
	        this.newPath = source["newPath"];
	        this.displayValue = source["displayValue"];
	        this.status = source["status"];
	        this.note = source["note"];
	    }
	}
	export class LegacyMappingReport {
	    mappedOk: LegacyFieldEntry[];
	    needsConfirm: LegacyFieldEntry[];
	    sensitive: LegacyFieldEntry[];
	    unsupported: LegacyFieldEntry[];
	    parseErrors: string[];
	
	    static createFrom(source: any = {}) {
	        return new LegacyMappingReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mappedOk = this.convertValues(source["mappedOk"], LegacyFieldEntry);
	        this.needsConfirm = this.convertValues(source["needsConfirm"], LegacyFieldEntry);
	        this.sensitive = this.convertValues(source["sensitive"], LegacyFieldEntry);
	        this.unsupported = this.convertValues(source["unsupported"], LegacyFieldEntry);
	        this.parseErrors = source["parseErrors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LegacyParseResult {
	    report: LegacyMappingReport;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new LegacyParseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.report = this.convertValues(source["report"], LegacyMappingReport);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MailDomainStatRow {
	    index: number;
	    domain: string;
	    veri: number;
	    live: number;
	    die: number;
	    rate: number;
	
	    static createFrom(source: any = {}) {
	        return new MailDomainStatRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.domain = source["domain"];
	        this.veri = source["veri"];
	        this.live = source["live"];
	        this.die = source["die"];
	        this.rate = source["rate"];
	    }
	}
	export class PhoneCountryInfo {
	    name: string;
	    countryCode: string;
	    phoneCode: string;
	    areaCode: string;
	
	    static createFrom(source: any = {}) {
	        return new PhoneCountryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.countryCode = source["countryCode"];
	        this.phoneCode = source["phoneCode"];
	        this.areaCode = source["areaCode"];
	    }
	}
	
	export class ProfileInfo {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class RegStatRow {
	    index: number;
	    platform: string;
	    success: number;
	    fail: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new RegStatRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.platform = source["platform"];
	        this.success = source["success"];
	        this.fail = source["fail"];
	        this.total = source["total"];
	    }
	}
	export class RegisterInput {
	    firstName: string;
	    lastName: string;
	    birthday: string;
	    gender: number;
	    phone: string;
	    password: string;
	    proxy: string;
	    userAgent: string;
	
	    static createFrom(source: any = {}) {
	        return new RegisterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.firstName = source["firstName"];
	        this.lastName = source["lastName"];
	        this.birthday = source["birthday"];
	        this.gender = source["gender"];
	        this.phone = source["phone"];
	        this.password = source["password"];
	        this.proxy = source["proxy"];
	        this.userAgent = source["userAgent"];
	    }
	}
	export class SettingsData {
	    general: GeneralConfig;
	    ip: IpConfig;
	
	    static createFrom(source: any = {}) {
	        return new SettingsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.general = this.convertValues(source["general"], GeneralConfig);
	        this.ip = this.convertValues(source["ip"], IpConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UAPoolStatus {
	    kind: string;
	    path: string;
	    count: number;
	    overrideActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UAPoolStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.count = source["count"];
	        this.overrideActive = source["overrideActive"];
	    }
	}
	export class UploadSiteSourceConfig {
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UploadSiteSourceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	    }
	}
	export class UploadSiteConfig {
	    reg: UploadSiteSourceConfig;
	    ver: UploadSiteSourceConfig;
	    code: string;
	    apiKey: string;
	    adminUsername: string;
	    adminPassword: string;
	    filterDuplicate: boolean;
	    delayCheckSec: number;
	    accPerBatch: number;
	    delayBetweenBatchSec: number;
	
	    static createFrom(source: any = {}) {
	        return new UploadSiteConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reg = this.convertValues(source["reg"], UploadSiteSourceConfig);
	        this.ver = this.convertValues(source["ver"], UploadSiteSourceConfig);
	        this.code = source["code"];
	        this.apiKey = source["apiKey"];
	        this.adminUsername = source["adminUsername"];
	        this.adminPassword = source["adminPassword"];
	        this.filterDuplicate = source["filterDuplicate"];
	        this.delayCheckSec = source["delayCheckSec"];
	        this.accPerBatch = source["accPerBatch"];
	        this.delayBetweenBatchSec = source["delayBetweenBatchSec"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UploadStats {
	    totalUploaded: number;
	    totalFailed: number;
	    pendingCount: number;
	    consecutiveFailures: number;
	    duplicateSkipped: number;
	    lastUploadAt: string;
	    lastErrorAt: string;
	    lastError: string;
	    lastRotateAt: string;
	    startedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalUploaded = source["totalUploaded"];
	        this.totalFailed = source["totalFailed"];
	        this.pendingCount = source["pendingCount"];
	        this.consecutiveFailures = source["consecutiveFailures"];
	        this.duplicateSkipped = source["duplicateSkipped"];
	        this.lastUploadAt = source["lastUploadAt"];
	        this.lastErrorAt = source["lastErrorAt"];
	        this.lastError = source["lastError"];
	        this.lastRotateAt = source["lastRotateAt"];
	        this.startedAt = source["startedAt"];
	    }
	}
	export class VerifyRunConfig {
	    accountIds: number[];
	    maxThreads: number;
	    verifyConfig: instagram.VerifyConfig;
	    outputPath: string;
	    proxy: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyRunConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountIds = source["accountIds"];
	        this.maxThreads = source["maxThreads"];
	        this.verifyConfig = this.convertValues(source["verifyConfig"], instagram.VerifyConfig);
	        this.outputPath = source["outputPath"];
	        this.proxy = source["proxy"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace instagram {
	
	export class AddInfoConfig {
	    enabled: boolean;
	    city: boolean;
	    hometown: boolean;
	    school: boolean;
	    college: boolean;
	    work: boolean;
	    relationship: boolean;
	    dataDir: string;
	    delayMs: number;
	
	    static createFrom(source: any = {}) {
	        return new AddInfoConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.city = source["city"];
	        this.hometown = source["hometown"];
	        this.school = source["school"];
	        this.college = source["college"];
	        this.work = source["work"];
	        this.relationship = source["relationship"];
	        this.dataDir = source["dataDir"];
	        this.delayMs = source["delayMs"];
	    }
	}
	export class VerifyConfig {
	    verifyEnabled: boolean;
	    mailProvider: string;
	    mailList: string;
	    checkLiveDieEnabled: boolean;
	    timeDelayCheck: number;
	    timeDelaySendCode: number;
	    delayConfirmEmail: number;
	    delayVeriReg: number;
	    sendAgainCode: boolean;
	    waitMailMs: number;
	    maxResend: number;
	    outputPath: string;
	    uaIphoneList: string;
	    zeusXApiKey: string;
	    zeusXAccountCode: string;
	    dvfbApiKey: string;
	    dvfbAccountType: string;
	    store1sApiKey: string;
	    store1sProductId: string;
	    mail30sApiKey: string;
	    mail30sProductSlug: string;
	    tempMailLolApiKey: string;
	    tempMailDomain: string;
	    muaMailApiKey: string;
	    muaMailProductId: string;
	    unlimitMailApiKey: string;
	    unlimitMailProductId: string;
	    sptMailApiKey: string;
	    sptMailServiceCode: string;
	    emailAPIInfoApiKey: string;
	    emailAPIInfoProductCode: string;
	    otpCheapApiKey: string;
	    otpCheapServiceId: string;
	    shopGmail9999ApiKey: string;
	    shopGmail9999Service: string;
	    rentGmailApiKey: string;
	    rentGmailPlatform: string;
	    otpCodesSmsApiKey: string;
	    otpCodesSmsServiceId: string;
	    wmemailApiKey: string;
	    wmemailCommodity: string;
	    priyoEmailApiKey: string;
	    otpHotmailPriority?: string;
	    tempMailToken?: string;
	    reUseEmail: boolean;
	    useEmailTime: number;
	    fmUserTmpMail: boolean;
	    deepFakeLocale: boolean;
	    useProxyTempMail: boolean;
	    useProxyGmail: boolean;
	    enable2fa: boolean;
	    addInfo?: AddInfoConfig;
	
	    static createFrom(source: any = {}) {
	        return new VerifyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.verifyEnabled = source["verifyEnabled"];
	        this.mailProvider = source["mailProvider"];
	        this.mailList = source["mailList"];
	        this.checkLiveDieEnabled = source["checkLiveDieEnabled"];
	        this.timeDelayCheck = source["timeDelayCheck"];
	        this.timeDelaySendCode = source["timeDelaySendCode"];
	        this.delayConfirmEmail = source["delayConfirmEmail"];
	        this.delayVeriReg = source["delayVeriReg"];
	        this.sendAgainCode = source["sendAgainCode"];
	        this.waitMailMs = source["waitMailMs"];
	        this.maxResend = source["maxResend"];
	        this.outputPath = source["outputPath"];
	        this.uaIphoneList = source["uaIphoneList"];
	        this.zeusXApiKey = source["zeusXApiKey"];
	        this.zeusXAccountCode = source["zeusXAccountCode"];
	        this.dvfbApiKey = source["dvfbApiKey"];
	        this.dvfbAccountType = source["dvfbAccountType"];
	        this.store1sApiKey = source["store1sApiKey"];
	        this.store1sProductId = source["store1sProductId"];
	        this.mail30sApiKey = source["mail30sApiKey"];
	        this.mail30sProductSlug = source["mail30sProductSlug"];
	        this.tempMailLolApiKey = source["tempMailLolApiKey"];
	        this.tempMailDomain = source["tempMailDomain"];
	        this.muaMailApiKey = source["muaMailApiKey"];
	        this.muaMailProductId = source["muaMailProductId"];
	        this.unlimitMailApiKey = source["unlimitMailApiKey"];
	        this.unlimitMailProductId = source["unlimitMailProductId"];
	        this.sptMailApiKey = source["sptMailApiKey"];
	        this.sptMailServiceCode = source["sptMailServiceCode"];
	        this.emailAPIInfoApiKey = source["emailAPIInfoApiKey"];
	        this.emailAPIInfoProductCode = source["emailAPIInfoProductCode"];
	        this.otpCheapApiKey = source["otpCheapApiKey"];
	        this.otpCheapServiceId = source["otpCheapServiceId"];
	        this.shopGmail9999ApiKey = source["shopGmail9999ApiKey"];
	        this.shopGmail9999Service = source["shopGmail9999Service"];
	        this.rentGmailApiKey = source["rentGmailApiKey"];
	        this.rentGmailPlatform = source["rentGmailPlatform"];
	        this.otpCodesSmsApiKey = source["otpCodesSmsApiKey"];
	        this.otpCodesSmsServiceId = source["otpCodesSmsServiceId"];
	        this.wmemailApiKey = source["wmemailApiKey"];
	        this.wmemailCommodity = source["wmemailCommodity"];
	        this.priyoEmailApiKey = source["priyoEmailApiKey"];
	        this.otpHotmailPriority = source["otpHotmailPriority"];
	        this.tempMailToken = source["tempMailToken"];
	        this.reUseEmail = source["reUseEmail"];
	        this.useEmailTime = source["useEmailTime"];
	        this.fmUserTmpMail = source["fmUserTmpMail"];
	        this.deepFakeLocale = source["deepFakeLocale"];
	        this.useProxyTempMail = source["useProxyTempMail"];
	        this.useProxyGmail = source["useProxyGmail"];
	        this.enable2fa = source["enable2fa"];
	        this.addInfo = this.convertValues(source["addInfo"], AddInfoConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace rent {
	
	export class CredPool {
	    Provider: string;
	
	    static createFrom(source: any = {}) {
	        return new CredPool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Provider = source["Provider"];
	    }
	}

}

