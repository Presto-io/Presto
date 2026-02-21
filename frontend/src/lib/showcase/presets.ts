// Mock data for showcase pages

export const mockTemplates = [
  { name: '公文模板', author: 'Presto 官方', keywords: ['政务', '公文'], installed: true, builtin: true },
  { name: '教案模板（实操）', author: 'Presto 官方', keywords: ['教育', '教案'], installed: true, builtin: true },
  { name: '会议纪要', author: 'Presto 官方', keywords: ['办公', '会议'], installed: true, builtin: false },
  { name: '学术论文', author: '社区贡献', keywords: ['学术', '论文'], installed: true, builtin: false },
  { name: '个人简历', author: '社区贡献', keywords: ['求职', '简历'], installed: true, builtin: false },
  { name: '合同协议', author: '社区贡献', keywords: ['法务', '合同'], installed: true, builtin: false },
  { name: '周报模板', author: '社区贡献', keywords: ['办公', '汇报'], installed: true, builtin: false },
];

export const mockFiles = [
  { name: '第一季度工作总结.md', template: 'gongwen', autoDetected: true },
  { name: '五年级数学教案.md', template: 'jiaoan-shicao', autoDetected: true },
  { name: '产品需求文档.md', template: 'gongwen', autoDetected: false },
  { name: '家长通知书.md', template: 'jiaoan-shicao', autoDetected: true },
  { name: '部门预算报告.md', template: 'gongwen', autoDetected: false },
  { name: '实验报告.md', template: null as string | null, autoDetected: false },
];

// Example markdown content for gongwen template
export const gongwenExample = `---
title: "关于开展2025年度安全生产专项检查工作的通知"
author: "安全生产管理处"
date: "2025-03-15"
signature: true
template: "gongwen"
---

各部门、各单位：

为进一步加强安全生产管理，落实安全生产责任制，根据《安全生产法》和上级主管部门要求，决定在全公司范围内开展2025年度安全生产专项检查工作。现将有关事项通知如下。

## 工作目标

全面排查安全生产隐患，建立健全安全管理制度，提高全员安全意识，确保全年安全生产事故"零发生"。

## 检查范围与内容

### 检查范围

本次专项检查覆盖公司所有生产经营场所，包括：

1. 各生产车间及仓储区域
2. 办公场所及公共区域
3. 在建工程项目现场

### 重点检查内容

- 安全生产责任制落实情况
- 消防设施设备完好情况
- 特种设备检验及操作人员持证上岗情况
- 危险化学品储存、使用管理情况
- **应急预案**的制定及演练情况

## 工作安排

### 自查自纠阶段

各部门、各单位对照检查标准，全面开展自查自纠，建立问题清单，制定整改措施。

### 集中检查阶段

由安全生产管理处牵头，组织相关部门成立联合检查组，对各单位进行全面检查。

### 整改落实阶段

针对检查中发现的问题，责任单位须在规定期限内完成整改，并将整改报告报送安全生产管理处。

## 工作要求

各部门、各单位要高度重视此次专项检查工作，主要负责人要亲自部署、亲自督办。对检查中发现的重大隐患，要立即整改；对不能立即整改的，要制定切实可行的整改方案，明确整改期限和责任人。

特此通知。
`;

// Example markdown content for jiaoan template
export const jiaoanExample = `---
template: "jiaoan-shicao"
---

## 教学活动设计——PLC 基本指令应用

### 认识 PLC 硬件——了解 PLC 的基本组成与接线方法

#### 活动一：PLC 硬件认知

##### 0.5H

PLC 的基本组成：CPU 模块、输入模块、输出模块、电源模块。

观察实训台上的 PLC 设备，识别各模块位置及功能。

展示 PLC 实物，讲解各模块的功能与作用。

实物展示、讲练结合

##### 0.5H

PLC 的外部接线：输入端子接线、输出端子接线、电源接线。

按照接线图完成 PLC 的外部接线，并进行通电测试。

示范接线步骤，巡回指导学生操作，纠正接线错误。

任务驱动、示范教学

### 编写梯形图程序——掌握基本逻辑指令的使用方法

#### 活动二：基本指令编程练习

##### 1H

基本逻辑指令：LD、LDI、OUT、AND、ANI、OR、ORI。定时器指令：TON、TOFF。计数器指令：CTU。

使用编程软件编写"启动-保持-停止"控制程序，完成程序的下载与调试。

讲解指令功能与梯形图编程规则，演示编程软件的基本操作。

讲练结合、项目教学

#### 活动三：综合应用练习

##### 1H

综合运用基本指令和定时器实现"延时启动"和"闪烁控制"功能。

根据控制要求编写梯形图，下载至 PLC 运行并验证功能。

布置任务要求，引导学生分析控制逻辑，点评学生的编程方案。

任务驱动、小组讨论
`;
