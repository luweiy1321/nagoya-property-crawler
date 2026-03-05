import streamlit as st
import pandas as pd
import plotly.express as px
import os
import psycopg2
from psycopg2.extras import RealDictCursor

# Page config
st.set_page_config(
    page_title="名古屋房产信息",
    page_icon="🏠",
    layout="wide"
)

# Database connection - support both local and cloud
def get_connection():
    # Try environment variables first (for Streamlit Cloud)
    if 'DB_HOST' in os.environ:
        return psycopg2.connect(
            host=os.environ.get('DB_HOST'),
            port=int(os.environ.get('DB_PORT', 5432)),
            user=os.environ.get('DB_USER'),
            password=os.environ.get('DB_PASSWORD'),
            dbname=os.environ.get('DB_NAME')
        )
    # Fallback to local database
    else:
        return psycopg2.connect(
            host="localhost",
            port=5432,
            user="lw",
            password="",
            dbname="nagoya_properties"
        )

# Load data
@st.cache_data(ttl=60)
def load_data():
    try:
        conn = get_connection()
        cur = conn.cursor(cursor_factory=RealDictCursor)

        cur.execute("""
            SELECT
                id,
                source,
                property_id,
                listing_type,
                title,
                price,
                price_display,
                address,
                area,
                floor,
                layout,
                building_type,
                construction_year,
                station_name,
                walking_minutes,
                detail_url,
                created_at
            FROM properties
            ORDER BY created_at DESC
        """)
        data = cur.fetchall()
        cur.close()
        conn.close()
        return pd.DataFrame(data)
    except Exception as e:
        st.error(f"数据库连接失败: {e}")
        return pd.DataFrame()

def get_stats():
    try:
        conn = get_connection()
        cur = conn.cursor(cursor_factory=RealDictCursor)

        cur.execute("""
            SELECT
                source,
                COUNT(*) as count,
                COUNT(CASE WHEN title != '' AND address != '' AND area > 0 THEN 1 END) as complete_count,
                AVG(CASE WHEN area > 0 THEN area END) as avg_area,
                AVG(CASE WHEN price > 0 THEN price END) as avg_price
            FROM properties
            GROUP BY source
        """)
        stats = cur.fetchall()
        cur.close()
        conn.close()
        return pd.DataFrame(stats)
    except Exception as e:
        st.error(f"统计数据获取失败: {e}")
        return pd.DataFrame()

# Main app
def main():
    st.title("🏠 名古屋房产信息爬虫系统")

    # Load data
    with st.spinner("加载数据..."):
        df = load_data()
        stats = get_stats()

    if df.empty:
        st.error("""
        ### 无法连接到数据库

        **本地运行:**
        ```bash
        streamlit run streamlit_app.py
        ```

        **部署到Streamlit Cloud:**
        需要配置云端数据库连接信息，请设置以下Secrets:
        - `DB_HOST`
        - `DB_PORT`
        - `DB_USER`
        - `DB_PASSWORD`
        - `DB_NAME`

        推荐使用免费云端数据库:
        - [Supabase](https://supabase.com) - 免费500MB
        - [Neon](https://neon.tech) - 免费无限制
        - [Railway](https://railway.app) - 免费$5/月额度
        """)
        return

    # Sidebar filters
    st.sidebar.header("筛选条件")

    # Source filter
    all_sources = ['全部'] + list(df['source'].unique())
    source_filter = st.sidebar.selectbox("数据源", all_sources)

    # Listing type filter
    all_types = ['全部'] + list(df['listing_type'].dropna().unique())
    type_filter = st.sidebar.selectbox("房源类型", all_types)

    # Area range
    min_area = float(df['area'].min()) if df['area'].min() > 0 else 0
    max_area = float(df['area'].max()) if df['area'].max() > 0 else 200
    area_range = st.sidebar.slider("面积范围 (㎡)", min_area, max_area, (min_area, max_area))

    # Price range
    min_price = 0
    max_price = float(df['price'].max()) if df['price'].max() > 0 else 50
    price_range = st.sidebar.slider("价格范围 (万円)", min_price, max_price, (min_price, max_price))

    # Apply filters
    filtered_df = df.copy()

    if source_filter != '全部':
        filtered_df = filtered_df[filtered_df['source'] == source_filter]

    if type_filter != '全部':
        filtered_df = filtered_df[filtered_df['listing_type'] == type_filter]

    filtered_df = filtered_df[
        (filtered_df['area'] >= area_range[0]) &
        (filtered_df['area'] <= area_range[1]) &
        (filtered_df['price'] >= price_range[0] * 10000) &
        (filtered_df['price'] <= price_range[1] * 10000)
    ]

    # Stats cards
    st.subheader("数据统计")
    col1, col2, col3, col4 = st.columns(4)

    with col1:
        st.metric("总房源数", len(filtered_df))
    with col2:
        complete_count = len(filtered_df[(filtered_df['title'] != '') & (filtered_df['address'] != '') & (filtered_df['area'] > 0)])
        st.metric("完整数据", complete_count)
    with col3:
        avg_area = filtered_df[filtered_df['area'] > 0]['area'].mean()
        st.metric("平均面积", f"{avg_area:.1f}㎡" if avg_area > 0 else "N/A")
    with col4:
        avg_price = filtered_df[filtered_df['price'] > 0]['price'].mean() / 10000
        st.metric("平均价格", f"{avg_price:.1f}万円" if avg_price > 0 else "N/A")

    # Charts
    col1, col2 = st.columns(2)

    with col1:
        st.subheader("按数据源分布")
        source_counts = filtered_df['source'].value_counts()
        fig_source = px.pie(values=source_counts.values, names=source_counts.index, title="数据源分布")
        st.plotly_chart(fig_source, use_container_width=True)

    with col2:
        st.subheader("面积分布")
        area_data = filtered_df[filtered_df['area'] > 0]['area']
        fig_area = px.histogram(area_data, bins=30, title="面积分布 (㎡)")
        st.plotly_chart(fig_area, use_container_width=True)

    # Property list
    st.subheader(f"房源列表 ({len(filtered_df)} 条)")

    # Display options
    display_option = st.radio("显示方式", ["卡片", "表格"], horizontal=True)

    if display_option == "卡片":
        # Card view
        cols_per_row = 3
        for i in range(0, len(filtered_df), cols_per_row):
            cols = st.columns(cols_per_row)
            for j, (_, row) in enumerate(filtered_df.iloc[i:i+cols_per_row].iterrows()):
                with cols[j]:
                    st.markdown(f"""
                    <div style="border: 1px solid #ddd; padding: 15px; border-radius: 10px; background: white;">
                        <h4 style="margin: 0 0 10px 0;">{row['title'] or '无标题'}</h4>
                        <p style="margin: 5px 0; color: #666;">📍 {row['address'] or '地址未知'}</p>
                        <p style="margin: 5px 0;">🏢 {row['source']} | {'出租' if row['listing_type'] == 'rent' else '出售'}</p>
                        <p style="margin: 5px 0;">📏 {row['area']:.2f}㎡  💰 {row['price_display'] or 'N/A'}</p>
                        <p style="margin: 5px 0;">🚪 {row['layout'] or 'N/A'}  🏗️ {row['floor'] or 'N/A'}</p>
                        {f'<a href="{row["detail_url"]}" target="_blank" style="color: #0066cc;">查看详情 →</a>' if row['detail_url'] else ''}
                    </div>
                    """, unsafe_allow_html=True)
    else:
        # Table view
        display_df = filtered_df[[
            'source', 'title', 'address', 'area', 'price_display',
            'layout', 'floor', 'station_name', 'walking_minutes'
        ]].copy()
        display_df.columns = ['数据源', '标题', '地址', '面积(㎡)', '价格', '户型', '楼层', '车站', '步行(分)']
        st.dataframe(display_df, use_container_width=True, height=400)

    # Detail view on click
    if len(filtered_df) > 0:
        st.subheader("房源详情")
        selected_id = st.selectbox(
            "选择房源查看详情",
            options=filtered_df['id'].tolist(),
            format_func=lambda x: f"{filtered_df[filtered_df['id']==x]['title'].values[0] or '无标题'} - {filtered_df[filtered_df['id']==x]['source'].values[0]}"
        )

        if selected_id:
            prop = filtered_df[filtered_df['id'] == selected_id].iloc[0]
            st.markdown(f"""
            <div style="border: 1px solid #ddd; padding: 20px; border-radius: 10px; background: #f9f9f9;">
                <h2>{prop['title'] or '无标题'}</h2>
                <p><strong>数据源:</strong> {prop['source']}</p>
                <p><strong>房源ID:</strong> {prop['property_id']}</p>
                <p><strong>类型:</strong> {'出租' if prop['listing_type'] == 'rent' else '出售'}</p>
                <p><strong>地址:</strong> {prop['address'] or '未知'}</p>
                <p><strong>价格:</strong> {prop['price_display'] or 'N/A'}</p>
                <p><strong>面积:</strong> {prop['area']:.2f}㎡</p>
                <p><strong>户型:</strong> {prop['layout'] or 'N/A'}</p>
                <p><strong>楼层:</strong> {prop['floor'] or 'N/A'}</p>
                <p><strong>建筑类型:</strong> {prop['building_type'] or 'N/A'}</p>
                <p><strong>建造年份:</strong> {prop['construction_year'] or 'N/A'}</p>
                <p><strong>最近车站:</strong> {prop['station_name'] or 'N/A'}</p>
                <p><strong>步行时间:</strong> {prop['walking_minutes'] or 'N/A'} 分钟</p>
                {f'<p><strong>详情链接:</strong> <a href="{prop["detail_url"]}" target="_blank">查看</a></p>' if prop['detail_url'] else ''}
            </div>
            """, unsafe_allow_html=True)

    # Refresh button
    if st.button("🔄 刷新数据"):
        st.cache_data.clear()
        st.rerun()

if __name__ == "__main__":
    main()
