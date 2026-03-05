/**
 * 名古屋房产爬虫系统 - 前端交互脚本
 */

// 等待 DOM 加载完成
document.addEventListener('DOMContentLoaded', function() {
    // 初始化各种功能
    initFilterForm();
    initPropertyCards();
    initImageGallery();
    initAPIEndpoints();
});

/**
 * 筛选表单处理
 */
function initFilterForm() {
    const filterForm = document.getElementById('filterForm');
    if (!filterForm) return;

    // 监听表单提交
    filterForm.addEventListener('submit', function(e) {
        // 可以在这里添加客户端验证
        const minPrice = filterForm.querySelector('[name="min_price"]');
        const maxPrice = filterForm.querySelector('[name="max_price"]');
        const minArea = filterForm.querySelector('[name="min_area"]');
        const maxArea = filterForm.querySelector('[name="max_area"]');

        // 验证价格范围
        if (minPrice && maxPrice && minPrice.value && maxPrice.value) {
            if (parseInt(minPrice.value) > parseInt(maxPrice.value)) {
                e.preventDefault();
                alert('最低价格不能高于最高价格');
                return;
            }
        }

        // 验证面积范围
        if (minArea && maxArea && minArea.value && maxArea.value) {
            if (parseFloat(minArea.value) > parseFloat(maxArea.value)) {
                e.preventDefault();
                alert('最小面积不能大于最大面积');
                return;
            }
        }
    });

    // 添加重置按钮功能
    const resetBtn = filterForm.querySelector('[type="reset"], .btn-reset');
    if (resetBtn) {
        resetBtn.addEventListener('click', function() {
            filterForm.reset();
            window.location.href = '/list';
        });
    }
}

/**
 * 房产卡片交互
 */
function initPropertyCards() {
    const cards = document.querySelectorAll('.property-card');
    cards.forEach(card => {
        // 添加收藏功能（可选）
        const favoriteBtn = card.querySelector('.btn-favorite');
        if (favoriteBtn) {
            favoriteBtn.addEventListener('click', function(e) {
                e.preventDefault();
                toggleFavorite(this, card);
            });
        }

        // 添加图片懒加载
        const images = card.querySelectorAll('img[data-src]');
        images.forEach(img => {
            img.setAttribute('src', img.getAttribute('data-src'));
            img.removeAttribute('data-src');
        });
    });
}

/**
 * 收藏功能
 */
function toggleFavorite(btn, card) {
    const propertyId = card.dataset.propertyId;
    const isFavorited = btn.classList.contains('active');

    if (isFavorited) {
        btn.classList.remove('active');
        btn.innerHTML = '<i class="bi bi-heart"></i> 收藏';
        // 可以在这里移除本地存储的收藏
        removeFavorite(propertyId);
    } else {
        btn.classList.add('active');
        btn.innerHTML = '<i class="bi bi-heart-fill"></i> 已收藏';
        // 可以在这里保存到本地存储
        saveFavorite(propertyId);
    }
}

/**
 * 保存收藏到本地存储
 */
function saveFavorite(propertyId) {
    let favorites = JSON.parse(localStorage.getItem('favorites') || '[]');
    if (!favorites.includes(propertyId)) {
        favorites.push(propertyId);
        localStorage.setItem('favorites', JSON.stringify(favorites));
    }
}

/**
 * 从本地存储移除收藏
 */
function removeFavorite(propertyId) {
    let favorites = JSON.parse(localStorage.getItem('favorites') || '[]');
    favorites = favorites.filter(id => id !== propertyId);
    localStorage.setItem('favorites', JSON.stringify(favorites));
}

/**
 * 图片画廊功能
 */
function initImageGallery() {
    const galleryContainer = document.querySelector('.gallery-container');
    if (!galleryContainer) return;

    // 灯箱效果
    const images = galleryContainer.querySelectorAll('img');
    images.forEach(img => {
        img.addEventListener('click', function() {
            openLightbox(this.src);
        });
    });
}

/**
 * 打开灯箱
 */
function openLightbox(src) {
    const lightbox = document.createElement('div');
    lightbox.className = 'lightbox';
    lightbox.innerHTML = `
        <div class="lightbox-overlay" onclick="this.parentElement.remove()"></div>
        <div class="lightbox-content">
            <img src="${src}" alt="预览图片">
            <button class="lightbox-close" onclick="this.closest('.lightbox').remove()">
                <i class="bi bi-x-lg"></i>
            </button>
        </div>
    `;
    document.body.appendChild(lightbox);
    document.body.style.overflow = 'hidden';
}

/**
 * API 端点工具函数
 */
function initAPIEndpoints() {
    // 可以在这里添加 API 调用的辅助函数
    window.API = {
        baseUrl: '/api',

        // 获取统计数据
        getStats: async function() {
            const response = await fetch(`${this.baseUrl}/stats`);
            return response.json();
        },

        // 获取房产列表
        getProperties: async function(filters = {}) {
            const params = new URLSearchParams(filters);
            const response = await fetch(`${this.baseUrl}/properties?${params}`);
            return response.json();
        },

        // 获取单个房产
        getProperty: async function(id) {
            const response = await fetch(`${this.baseUrl}/properties/${id}`);
            return response.json();
        }
    };
}

/**
 * 分页助手函数
 */
function buildPaginationUrl(page, filters) {
    const params = new URLSearchParams(window.location.search);
    params.set('page', page);
    return `${window.location.pathname}?${params.toString()}`;
}

/**
 * 格式化价格显示
 */
function formatPrice(price, listingType) {
    if (listingType === 'rent') {
        return (price / 10000).toFixed(1) + ' 万円/月';
    } else {
        return Math.floor(price / 10000).toLocaleString() + ' 万円';
    }
}

/**
 * 格式化面积显示
 */
function formatArea(area) {
    return area.toFixed(2) + ' ㎡';
}

/**
 * 显示加载状态
 */
function showLoading(element) {
    const spinner = document.createElement('div');
    spinner.className = 'spinner-border spinner-border-sm ms-2';
    spinner.setAttribute('role', 'status');
    element.appendChild(spinner);
    return spinner;
}

/**
 * 隐藏加载状态
 */
function hideLoading(spinner) {
    if (spinner && spinner.parentNode) {
        spinner.parentNode.removeChild(spinner);
    }
}

/**
 * 显示提示消息
 */
function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">${message}</div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;

    let container = document.querySelector('.toast-container');
    if (!container) {
        container = document.createElement('div');
        container.className = 'toast-container position-fixed bottom-0 end-0 p-3';
        document.body.appendChild(container);
    }

    container.appendChild(toast);

    const bsToast = new bootstrap.Toast(toast, { delay: 3000 });
    bsToast.show();

    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
}

/**
 * 防抖函数
 */
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

/**
 * 节流函数
 */
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// 导出工具函数供全局使用
window.Utils = {
    formatPrice,
    formatArea,
    showLoading,
    hideLoading,
    showToast,
    debounce,
    throttle
};
