properties([
    parameters([
        booleanParam(name: 'BUILD_ALL', defaultValue: true, description: 'Bangun semua service (Default)'),
        booleanParam(name: 'BUILD_ORDER_TARIFF', defaultValue: false, description: 'Hanya bangun Order Tariff Service'),
        booleanParam(name: 'BUILD_SHIPPING', defaultValue: false, description: 'Hanya bangun Shipping Service'),
        booleanParam(name: 'BUILD_WAREHOUSE', defaultValue: false, description: 'Hanya bangun Warehouse & Inventory Service'),
        booleanParam(name: 'BUILD_TRACKING', defaultValue: false, description: 'Hanya bangun Tracking & Log Event Service'),
        booleanParam(name: 'BUILD_NOTIFICATION', defaultValue: false, description: 'Hanya bangun Notification & Messaging Service')
    ])
])

def getChangedServices() {
    def changed = []
    try {
        // Deteksi file yang berubah pada commit changeset
        def changeLogSets = currentBuild.changeSets
        for (int i = 0; i < changeLogSets.size(); i++) {
            def entries = changeLogSets[i].items
            for (int j = 0; j < entries.length; j++) {
                def entry = entries[j]
                def paths = entry.affectedPaths
                for (int k = 0; k < paths.size(); k++) {
                    def path = paths[k]
                    if (path.startsWith('order-tariff-service/')) {
                        changed.add('order-tariff')
                    } else if (path.startsWith('shipping-service/')) {
                        changed.add('shipping')
                    } else if (path.startsWith('warehouse-and-inventory-service/')) {
                        changed.add('warehouse')
                    } else if (path.startsWith('tracking-and-logevent-service/')) {
                        changed.add('tracking')
                    } else if (path.startsWith('notification-and-messaging-service/')) {
                        changed.add('notification')
                    }
                }
            }
        }
    } catch (Exception e) {
        echo "Gagal mendeteksi perubahan Git: ${e.message}. Menjalankan semua service sebagai fallback."
        return ['order-tariff', 'shipping', 'warehouse', 'tracking', 'notification']
    }
    return changed.unique()
}

node {
    def gitBranch = env.BRANCH_NAME ?: 'develop'
    def gitUrl = 'https://github.com/madgeer/microservce-papiton-express.git'

    stage('Checkout Root') {
        echo "Checking out repository from branch ${gitBranch}..."
        git url: gitUrl, branch: gitBranch
    }

    // Menentukan daftar service yang akan dieksekusi
    def servicesToBuild = []

    // 1. Jika build dipicu secara manual via Jenkins UI menggunakan parameter
    def isManual = params.BUILD_ORDER_TARIFF || params.BUILD_SHIPPING || params.BUILD_WAREHOUSE || params.BUILD_TRACKING || params.BUILD_NOTIFICATION
    
    if (isManual) {
        if (params.BUILD_ALL) {
            servicesToBuild = ['order-tariff', 'shipping', 'warehouse', 'tracking', 'notification']
        } else {
            if (params.BUILD_ORDER_TARIFF) servicesToBuild.add('order-tariff')
            if (params.BUILD_SHIPPING) servicesToBuild.add('shipping')
            if (params.BUILD_WAREHOUSE) servicesToBuild.add('warehouse')
            if (params.BUILD_TRACKING) servicesToBuild.add('tracking')
            if (params.BUILD_NOTIFICATION) servicesToBuild.add('notification')
        }
    } else {
        // 2. Jika dipicu otomatis (webhook/push commit), deteksi berdasarkan folder yang berubah
        echo "Mendeteksi perubahan folder service untuk menentukan pipeline yang berjalan..."
        servicesToBuild = getChangedServices()
        if (servicesToBuild.isEmpty()) {
            echo "Tidak ada perubahan spesifik service terdeteksi atau ini build pertama. Menjalankan seluruh service."
            servicesToBuild = ['order-tariff', 'shipping', 'warehouse', 'tracking', 'notification']
        }
    }

    echo "Pipeline yang akan dijalankan untuk service: ${servicesToBuild}"

    // Struktur paralel untuk mengeksekusi pipeline masing-masing service secara terisolasi
    def parallelStages = [:]

    if (servicesToBuild.contains('order-tariff')) {
        parallelStages['Order Tariff Pipeline'] = {
            echo "Starting pipeline for Order Tariff Service..."
            load 'order-tariff-service/Jenkinsfile'
        }
    }

    if (servicesToBuild.contains('shipping')) {
        parallelStages['Shipping Pipeline'] = {
            echo "Starting pipeline for Shipping Service..."
            load 'shipping-service/Jenkinsfile'
        }
    }

    if (servicesToBuild.contains('warehouse')) {
        parallelStages['Warehouse Pipeline'] = {
            echo "Starting pipeline for Warehouse & Inventory Service..."
            load 'warehouse-and-inventory-service/Jenkinsfile'
        }
    }

    if (servicesToBuild.contains('tracking')) {
        parallelStages['Tracking Pipeline'] = {
            echo "Starting pipeline for Tracking & Log Event Service..."
            load 'tracking-and-logevent-service/Jenkinsfile'
        }
    }

    if (servicesToBuild.contains('notification')) {
        parallelStages['Notification Pipeline'] = {
            echo "Starting pipeline for Notification & Messaging Service..."
            load 'notification-and-messaging-service/Jenkinsfile'
        }
    }

    // Jalankan seluruh service terpilih secara paralel
    if (!parallelStages.isEmpty()) {
        parallel parallelStages
    } else {
        echo "Tidak ada pipeline service yang perlu dijalankan."
    }
}
